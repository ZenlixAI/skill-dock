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

interface RegistryPackageJson {
  name?: unknown;
  version?: unknown;
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

async function verifyPackageMetadata(target: InstallTarget, packageJsonPath: string): Promise<void> {
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
  target: InstallTarget
): Promise<DownloadedRegistrySkill> {
  const workingDir = await mkdtemp(join(tmpdir(), 'skills-registry-'));
  const archivePath = join(workingDir, 'bundle.tgz');
  const extractDir = join(workingDir, 'extract');

  const cleanup = async () => {
    await rm(workingDir, { recursive: true, force: true });
  };

  try {
    const response = await fetch(target.tarballUrl, {
      headers: { accept: 'application/octet-stream' },
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
        name: target.packageName,
        path: packageDir,
      },
      cleanup,
    };
  } catch (error) {
    await cleanup();
    throw error;
  }
}
