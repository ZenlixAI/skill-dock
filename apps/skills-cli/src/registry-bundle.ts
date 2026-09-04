import { spawnSync } from 'child_process';
import { existsSync } from 'fs';
import { createHash } from 'crypto';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'fs/promises';
import { tmpdir } from 'os';
import { join } from 'path';
import { parseSkillMd } from './skills.ts';
import type { InstallTarget } from './catalog.ts';
import type { Skill } from './types.ts';

export interface DownloadedRegistrySkill {
  extractedPath: string;
  skill: Skill;
  cleanup: () => Promise<void>;
}

interface DownloadRegistrySkillBundleOptions {
  cwd?: string;
  userConfigPath?: string;
}

interface RegistryPackageJson {
  name?: unknown;
  version?: unknown;
}

interface NpmRcAuthEntry {
  host: string;
  pathPrefix: string;
  token: string;
}

function packageBasename(packageName: string): string {
  const parts = packageName.split('/');
  return parts[parts.length - 1] || packageName;
}

function resolveUserConfigPath(options: DownloadRegistrySkillBundleOptions): string | undefined {
  if (options.userConfigPath && options.userConfigPath.trim() !== '') {
    return options.userConfigPath.trim();
  }

  const envConfigPath = process.env.NPM_CONFIG_USERCONFIG?.trim();
  if (envConfigPath) {
    return envConfigPath;
  }

  if (options.cwd && options.cwd.trim() !== '') {
    return join(options.cwd, '.npmrc');
  }

  return undefined;
}

function parseNpmRcAuthEntries(content: string): NpmRcAuthEntry[] {
  const entries: NpmRcAuthEntry[] = [];

  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#') || line.startsWith(';')) {
      continue;
    }

    const tokenMarker = ':_authToken=';
    const markerIndex = line.indexOf(tokenMarker);
    if (!line.startsWith('//') || markerIndex < 0) {
      continue;
    }

    const registryPart = line.slice(2, markerIndex);
    const token = line.slice(markerIndex + tokenMarker.length).trim();
    if (!registryPart || token === '') {
      continue;
    }

    const slashIndex = registryPart.indexOf('/');
    const host = slashIndex >= 0 ? registryPart.slice(0, slashIndex) : registryPart;
    if (!host) {
      continue;
    }

    const pathPrefixRaw = slashIndex >= 0 ? registryPart.slice(slashIndex) : '/';
    const normalizedPathPrefix = pathPrefixRaw.startsWith('/') ? pathPrefixRaw : `/${pathPrefixRaw}`;
    const pathPrefix = normalizedPathPrefix === '/' ? '/' : `${normalizedPathPrefix.replace(/\/+$/, '')}/`;

    entries.push({
      host,
      pathPrefix,
      token,
    });
  }

  return entries;
}

function selectNpmRcAuth(targetURL: string, content: string): string | undefined {
  let parsed: URL;
  try {
    parsed = new URL(targetURL);
  } catch {
    return undefined;
  }

  const entries = parseNpmRcAuthEntries(content)
    .filter((entry) => entry.host === parsed.host && parsed.pathname.startsWith(entry.pathPrefix))
    .sort((left, right) => right.pathPrefix.length - left.pathPrefix.length);

  const token = entries[0]?.token;
  if (!token) {
    return undefined;
  }

  return `Bearer ${token}`;
}

function resolveRegistryAccessMode(target: InstallTarget): string {
  return String(target.registryAccessMode || '').trim().toLowerCase();
}

function loadRegistryAuthorization(
  target: InstallTarget,
  options: DownloadRegistrySkillBundleOptions = {}
): Promise<string | undefined> {
  if (resolveRegistryAccessMode(target) !== 'direct') {
    return Promise.resolve(undefined);
  }

  const configPath = resolveUserConfigPath(options);
  if (!configPath) {
    return Promise.resolve(undefined);
  }

  return readFile(configPath, 'utf-8')
    .then((content) => selectNpmRcAuth(target.tarballUrl, content))
    .catch(() => undefined);
}

function verifyTarballIntegrity(target: InstallTarget, tarball: Buffer): void {
  const [algorithm, expectedDigest] = target.integrity.split('-', 2);
  if (!algorithm || !expectedDigest) {
    throw new Error('Resolved bundle integrity string is invalid');
  }

  const actualDigest = createHash(algorithm).update(tarball).digest('base64');
  if (actualDigest !== expectedDigest) {
    throw new Error('Resolved bundle failed integrity verification');
  }
}

async function verifyManifestDigest(target: InstallTarget, skillMdPath: string): Promise<void> {
  const skillMd = await readFile(skillMdPath);
  const actualDigest = `sha256:${createHash('sha256').update(skillMd).digest('hex')}`;
  if (actualDigest !== target.skillManifestDigest) {
    throw new Error('Resolved bundle failed manifest digest verification');
  }
}

async function verifyPackageMetadata(
  target: InstallTarget,
  packageJsonPath: string
): Promise<void> {
  let parsed: RegistryPackageJson;
  try {
    parsed = JSON.parse(await readFile(packageJsonPath, 'utf-8')) as RegistryPackageJson;
  } catch {
    throw new Error('Resolved bundle package metadata does not match the install target');
  }

  if (parsed.name !== target.packageName || parsed.version !== target.version) {
    throw new Error('Resolved bundle package metadata does not match the install target');
  }
}

export async function downloadRegistrySkillBundle(
  target: InstallTarget,
  options: DownloadRegistrySkillBundleOptions = {}
): Promise<DownloadedRegistrySkill> {
  const workingDir = await mkdtemp(join(tmpdir(), 'skills-registry-'));
  const archivePath = join(workingDir, 'bundle.tgz');
  const extractDir = join(workingDir, 'extract');

  const cleanup = async () => {
    await rm(workingDir, { recursive: true, force: true });
  };

  try {
    const authorization = await loadRegistryAuthorization(target, options);
    const headers: Record<string, string> = {
      accept: 'application/octet-stream',
    };
    if (authorization) {
      headers.Authorization = authorization;
    }

    const response = await fetch(target.tarballUrl, {
      headers,
    });

    if (!response.ok) {
      throw new Error(`Failed to download tarball: ${response.status} ${response.statusText}`);
    }

    const buffer = Buffer.from(await response.arrayBuffer());
    verifyTarballIntegrity(target, buffer);
    await writeFile(archivePath, buffer);
    await mkdir(extractDir, { recursive: true });

    const result = spawnSync('tar', ['-xzf', archivePath, '-C', extractDir], {
      encoding: 'utf-8',
    });

    if (result.status !== 0) {
      throw new Error(result.stderr || result.stdout || 'Failed to extract skill bundle');
    }

    const packageDir = join(extractDir, 'package');
    const skillMdPath = join(packageDir, 'SKILL.md');
    const packageJsonPath = join(packageDir, 'package.json');

    if (!existsSync(packageDir) || !existsSync(skillMdPath) || !existsSync(packageJsonPath)) {
      throw new Error('Resolved bundle does not contain a root SKILL.md');
    }

    await verifyManifestDigest(target, skillMdPath);
    await verifyPackageMetadata(target, packageJsonPath);

    const parsed = await parseSkillMd(skillMdPath, { includeInternal: true });
    if (!parsed) {
      throw new Error('Resolved bundle contains an invalid SKILL.md');
    }

    return {
      extractedPath: packageDir,
      skill: {
        ...parsed,
        packageName: target.packageName,
        installName: packageBasename(target.packageName),
        path: packageDir,
      },
      cleanup,
    };
  } catch (error) {
    await cleanup();
    throw error;
  }
}
