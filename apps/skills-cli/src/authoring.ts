import { existsSync } from 'fs';
import { readFile, readdir, stat } from 'fs/promises';
import { createHash } from 'crypto';
import { join } from 'path';
import { parseFrontmatter } from './frontmatter.ts';

export const PACKAGE_NAME_RE = /^(?:@[a-z0-9][a-z0-9._-]*\/)?[a-z0-9][a-z0-9._-]*$/;
export const EXACT_VERSION_RE =
  /^\d+\.\d+\.\d+(?:-[0-9A-Za-z]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/;

const CAPABILITY_DIRS = ['commands', 'agents', 'hooks', 'scripts', 'docs'];
const CAPABILITY_FILES = ['plugin.json', '.mcp.json'];

export interface SkillAuthoringInspection {
  ok: boolean;
  directory: string;
  skillMdPath: string;
  packageJsonPath: string;
  displayName?: string;
  description?: string;
  packageName?: string;
  version?: string;
  publishRegistry?: string;
  sourceType?: string;
  components: string[];
  errors: string[];
}

interface PackageJsonShape {
  name?: unknown;
  version?: unknown;
  description?: unknown;
  publishConfig?: {
    registry?: unknown;
  };
  skill?: {
    sourceType?: unknown;
  };
}

export function isPackageName(value: string): boolean {
  return PACKAGE_NAME_RE.test(value);
}

export function hasSkillPackageScope(value: string): boolean {
  if (!isPackageName(value) || !value.startsWith('@')) {
    return false;
  }

  const slashIndex = value.indexOf('/');
  if (slashIndex <= 1) {
    return false;
  }

  const scope = value.slice(1, slashIndex);
  return scope.endsWith('.skill');
}

export function isExactVersion(value: string): boolean {
  return EXACT_VERSION_RE.test(value);
}

async function readPackageJson(packageJsonPath: string): Promise<PackageJsonShape> {
  const raw = await readFile(packageJsonPath, 'utf-8');
  return JSON.parse(raw) as PackageJsonShape;
}

async function detectComponents(directory: string): Promise<string[]> {
  const components: string[] = [];

  async function hasPublishableContent(path: string): Promise<boolean> {
    try {
      const entries = await readdir(path, { withFileTypes: true });
      for (const entry of entries) {
        if (entry.isFile()) {
          return true;
        }
        if (entry.isDirectory() && (await hasPublishableContent(join(path, entry.name)))) {
          return true;
        }
      }
    } catch {
      return false;
    }
    return false;
  }

  await Promise.all(
    CAPABILITY_DIRS.map(async (name) => {
      try {
        const entry = await stat(join(directory, name));
        if (entry.isDirectory() && (await hasPublishableContent(join(directory, name)))) {
          components.push(name);
        }
      } catch {
        // Ignore missing directories.
      }
    })
  );

  await Promise.all(
    CAPABILITY_FILES.map(async (name) => {
      try {
        const entry = await stat(join(directory, name));
        if (entry.isFile()) {
          components.push(name);
        }
      } catch {
        // Ignore missing files.
      }
    })
  );

  return components.sort();
}

export async function inspectSkillDirectory(directory: string): Promise<SkillAuthoringInspection> {
  const skillMdPath = join(directory, 'SKILL.md');
  const packageJsonPath = join(directory, 'package.json');
  const errors: string[] = [];

  if (!existsSync(skillMdPath)) {
    errors.push('SKILL.md is required');
  }

  if (!existsSync(packageJsonPath)) {
    errors.push('package.json is required');
  }

  let packageName: string | undefined;
  let version: string | undefined;
  let publishRegistry: string | undefined;
  let sourceType: string | undefined;
  let displayName: string | undefined;
  let description: string | undefined;

  if (existsSync(skillMdPath)) {
    try {
      const raw = await readFile(skillMdPath, 'utf-8');
      const { data } = parseFrontmatter(raw);

      if (typeof data.name !== 'string' || data.name.trim() === '') {
        errors.push('SKILL.md frontmatter name is required');
      } else {
        displayName = data.name.trim();
      }

      if (typeof data.description !== 'string' || data.description.trim() === '') {
        errors.push('SKILL.md frontmatter description is required');
      } else {
        description = data.description.trim();
      }
    } catch {
      errors.push('SKILL.md could not be read');
    }
  }

  if (existsSync(packageJsonPath)) {
    try {
      const pkg = await readPackageJson(packageJsonPath);

      if (typeof pkg.name !== 'string' || pkg.name.trim() === '') {
        errors.push('package.json name is required');
      } else if (!isPackageName(pkg.name)) {
        errors.push('package.json name must be a valid package_name');
      } else if (!hasSkillPackageScope(pkg.name)) {
        errors.push('package.json name must use a scoped @<company>.skill/<skill-name> package name');
      } else {
        packageName = pkg.name;
      }

      if (typeof pkg.version !== 'string' || pkg.version.trim() === '') {
        errors.push('package.json version is required');
      } else if (!isExactVersion(pkg.version)) {
        errors.push('package.json version must be an exact semver');
      } else {
        version = pkg.version;
      }

      if (
        pkg.publishConfig?.registry !== undefined &&
        typeof pkg.publishConfig.registry !== 'string'
      ) {
        errors.push('package.json publishConfig.registry must be a string');
      } else if (typeof pkg.publishConfig?.registry === 'string') {
        publishRegistry = pkg.publishConfig.registry;
      }

      if (pkg.skill?.sourceType !== undefined && typeof pkg.skill.sourceType !== 'string') {
        errors.push('package.json skill.sourceType must be a string');
      } else if (typeof pkg.skill?.sourceType === 'string' && pkg.skill.sourceType.trim() !== '') {
        sourceType = pkg.skill.sourceType.trim();
      }
    } catch {
      errors.push('package.json must contain valid JSON');
    }
  }

  return {
    ok: errors.length === 0,
    directory,
    skillMdPath,
    packageJsonPath,
    displayName,
    description,
    packageName,
    version,
    publishRegistry,
    ...(sourceType ? { sourceType } : {}),
    components: await detectComponents(directory),
    errors,
  };
}

export async function computeManifestDigest(directory: string): Promise<string> {
  const content = await readFile(join(directory, 'SKILL.md'));
  return `sha256:${createHash('sha256').update(content).digest('hex')}`;
}
