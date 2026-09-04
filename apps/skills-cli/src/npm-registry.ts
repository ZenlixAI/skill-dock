import { spawnSync } from 'child_process';
import { existsSync } from 'fs';
import { createHash } from 'crypto';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'fs/promises';
import { tmpdir } from 'os';
import { join } from 'path';

export interface NpmRegistrySearchResult {
  packageName: string;
  displayName: string;
  description: string;
  latestVersion?: string;
  originRegistryUrl: string;
}

interface NpmRcAuthEntry {
  host: string;
  pathPrefix: string;
  token: string;
}

interface NpmPackageVersionMetadata {
  name?: string;
  version?: string;
  description?: string;
  dist?: {
    tarball?: string;
    integrity?: string;
    shasum?: string;
  };
  skill?: {
    displayName?: string;
    sourceType?: string;
    validationError?: string;
  };
}

interface NpmPackageMetadata {
  name?: string;
  description?: string;
  'dist-tags'?: Record<string, string>;
  versions?: Record<string, NpmPackageVersionMetadata>;
}

export interface NpmPackageSelection {
  packageName: string;
  version: string;
  tarballUrl: string;
  description: string;
  displayName: string;
  registryUrl: string;
}

export interface DownloadedNpmPackage {
  extractedPath: string;
  selection: NpmPackageSelection;
  cleanup: () => Promise<void>;
}

export interface NpmRegistryRequestOptions {
  cwd?: string;
  userConfigPath?: string;
}

export function resolveNpmRegistryUrl(explicit?: string): string {
  const value =
    explicit?.trim() ||
    process.env.npm_config_registry?.trim() ||
    process.env.NPM_CONFIG_REGISTRY?.trim() ||
    'https://registry.npmjs.org';
  return value.replace(/\/+$/, '');
}

function resolveUserConfigPath(options: NpmRegistryRequestOptions = {}): string | undefined {
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

    entries.push({ host, pathPrefix, token });
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

async function loadRegistryAuthorization(
  targetUrl: string,
  options: NpmRegistryRequestOptions = {}
): Promise<string | undefined> {
  const configPath = resolveUserConfigPath(options);
  if (!configPath) {
    return undefined;
  }

  return readFile(configPath, 'utf-8')
    .then((content) => selectNpmRcAuth(targetUrl, content))
    .catch(() => undefined);
}

function encodePackageName(packageName: string): string {
  return encodeURIComponent(packageName).replace(/%2F/g, '%2F');
}

function packageBasename(packageName: string): string {
  const parts = packageName.split('/');
  return parts[parts.length - 1] || packageName;
}

function resolveDisplayName(packageName: string, metadata: NpmPackageVersionMetadata, packageMetadata: NpmPackageMetadata): string {
  if (typeof metadata.skill?.displayName === 'string' && metadata.skill.displayName.trim() !== '') {
    return metadata.skill.displayName.trim();
  }
  if (typeof packageMetadata.name === 'string' && packageMetadata.name.trim() !== '') {
    return packageMetadata.name.trim();
  }
  const base = packageBasename(packageName);
  return base
    .split('-')
    .filter(Boolean)
    .map((word) => word[0]!.toUpperCase() + word.slice(1))
    .join(' ');
}

function resolveDescription(metadata: NpmPackageVersionMetadata, packageMetadata: NpmPackageMetadata): string {
  if (typeof metadata.description === 'string' && metadata.description.trim() !== '') {
    return metadata.description.trim();
  }
  if (typeof packageMetadata.description === 'string' && packageMetadata.description.trim() !== '') {
    return packageMetadata.description.trim();
  }
  return '';
}

async function fetchJson<T>(url: string, options: NpmRegistryRequestOptions = {}): Promise<T> {
  const authorization = await loadRegistryAuthorization(url, options);
  const headers: Record<string, string> = {
    accept: 'application/json',
  };
  if (authorization) {
    headers.Authorization = authorization;
  }

  const response = await fetch(url, { headers });
  if (!response.ok) {
    throw new Error(`Failed to load npm registry data: ${response.status} ${response.statusText}`);
  }
  return (await response.json()) as T;
}

async function fetchBinary(url: string, options: NpmRegistryRequestOptions = {}): Promise<Buffer> {
  const authorization = await loadRegistryAuthorization(url, options);
  const headers: Record<string, string> = {
    accept: 'application/octet-stream',
  };
  if (authorization) {
    headers.Authorization = authorization;
  }

  const response = await fetch(url, { headers });
  if (!response.ok) {
    throw new Error(`Failed to download npm tarball: ${response.status} ${response.statusText}`);
  }
  return Buffer.from(await response.arrayBuffer());
}

export async function searchNpmRegistrySkills(
  query: string,
  options: { registry?: string; size?: number } = {}
): Promise<NpmRegistrySearchResult[]> {
  const registryUrl = resolveNpmRegistryUrl(options.registry);
  const searchUrl = new URL('/-/v1/search', registryUrl + '/');
  searchUrl.searchParams.set('text', query);
  searchUrl.searchParams.set('size', String(options.size ?? 10));

  const payload = await fetchJson<{
    objects?: Array<{
      package?: {
        name?: string;
        version?: string;
        description?: string;
      };
    }>;
  }>(searchUrl.toString(), { cwd: process.cwd() });

  return (payload.objects || []).flatMap((entry) => {
    const pkg = entry.package;
    if (!pkg?.name) {
      return [];
    }
    return [
      {
        packageName: pkg.name,
        displayName: pkg.name,
        description: pkg.description || '',
        ...(pkg.version ? { latestVersion: pkg.version } : {}),
        originRegistryUrl: registryUrl,
      },
    ];
  });
}


export async function fetchNpmPackageSelection(
  packageName: string,
  options: { registry?: string; version?: string } = {}
): Promise<NpmPackageSelection> {
  const registryUrl = resolveNpmRegistryUrl(options.registry);
  const packageUrl = `${registryUrl}/${encodePackageName(packageName)}`;
  const metadata = await fetchJson<NpmPackageMetadata>(packageUrl, { cwd: process.cwd() });

  const requestedVersion = options.version?.trim();
  const version = requestedVersion || metadata['dist-tags']?.latest;
  if (!version) {
    throw new Error(`No version could be resolved for ${packageName}`);
  }

  const versionMetadata = metadata.versions?.[version];
  if (!versionMetadata) {
    throw new Error(`Version ${version} was not found for ${packageName}`);
  }
  const tarballUrl = versionMetadata.dist?.tarball;
  if (!tarballUrl) {
    throw new Error(`Version ${version} for ${packageName} does not expose a tarball URL`);
  }

  return {
    packageName,
    version,
    tarballUrl,
    description: resolveDescription(versionMetadata, metadata),
    displayName: resolveDisplayName(packageName, versionMetadata, metadata),
    registryUrl,
  };
}

export async function downloadAndExtractNpmPackage(
  selection: NpmPackageSelection,
  options: NpmRegistryRequestOptions = {}
): Promise<DownloadedNpmPackage> {
  const workingDir = await mkdtemp(join(tmpdir(), 'skills-npm-import-'));
  const archivePath = join(workingDir, 'bundle.tgz');
  const extractDir = join(workingDir, 'extract');

  const cleanup = async () => {
    await rm(workingDir, { recursive: true, force: true });
  };

  try {
    const tarball = await fetchBinary(selection.tarballUrl, options);
    await writeFile(archivePath, tarball);
    await mkdir(extractDir, { recursive: true });

    const result = spawnSync('tar', ['-xzf', archivePath, '-C', extractDir], {
      encoding: 'utf-8',
    });
    if (result.status !== 0) {
      throw new Error(result.stderr || result.stdout || 'Failed to extract npm package tarball');
    }

    const packageDir = join(extractDir, 'package');
    const extractedPath = existsSync(packageDir) ? packageDir : extractDir;

    return {
      extractedPath,
      selection,
      cleanup,
    };
  } catch (error) {
    await cleanup();
    throw error;
  }
}

export function sha256Hex(content: string | Buffer): string {
  return createHash('sha256').update(content).digest('hex');
}
