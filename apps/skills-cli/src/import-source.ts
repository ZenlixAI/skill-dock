import { existsSync } from 'fs';
import { readFile } from 'fs/promises';
import { join } from 'path';
import {
  hasSkillPackageScope,
  isExactVersion,
  isPackageName,
} from './authoring.ts';
import { parseFrontmatter } from './frontmatter.ts';

export interface ImportSourceInspection {
  directory: string;
  skillMdPath: string;
  packageJsonPath: string;
  displayName?: string;
  description?: string;
  packageName?: string;
  version?: string;
  publishRegistry?: string;
  sourceType?: string;
  errors: string[];
}

interface PackageJsonShape {
  name?: unknown;
  version?: unknown;
  publishConfig?: {
    registry?: unknown;
  };
  skill?: {
    sourceType?: unknown;
  };
}

async function readPackageJson(packageJsonPath: string): Promise<PackageJsonShape> {
  const raw = await readFile(packageJsonPath, 'utf-8');
  return JSON.parse(raw) as PackageJsonShape;
}

export async function inspectImportSourceDirectory(directory: string): Promise<ImportSourceInspection> {
  const skillMdPath = join(directory, 'SKILL.md');
  const packageJsonPath = join(directory, 'package.json');
  const errors: string[] = [];

  let displayName: string | undefined;
  let description: string | undefined;
  let packageName: string | undefined;
  let version: string | undefined;
  let publishRegistry: string | undefined;
  let sourceType: string | undefined;

  if (!existsSync(skillMdPath)) {
    errors.push('SKILL.md is required');
  } else {
    try {
      const raw = await readFile(skillMdPath, 'utf-8');
      const { data } = parseFrontmatter(raw);
      if (typeof data.name === 'string' && data.name.trim() !== '') {
        displayName = data.name.trim();
      } else {
        errors.push('SKILL.md frontmatter name is required');
      }
      if (typeof data.description === 'string' && data.description.trim() !== '') {
        description = data.description.trim();
      } else {
        errors.push('SKILL.md frontmatter description is required');
      }
    } catch {
      errors.push('SKILL.md could not be read');
    }
  }

  if (existsSync(packageJsonPath)) {
    try {
      const pkg = await readPackageJson(packageJsonPath);
      if (typeof pkg.name === 'string' && pkg.name.trim() !== '') {
        if (!isPackageName(pkg.name) || !hasSkillPackageScope(pkg.name)) {
          errors.push('package.json name must use a scoped @<company>.skill/<skill-name> package name');
        } else {
          packageName = pkg.name;
        }
      }
      if (typeof pkg.version === 'string' && pkg.version.trim() !== '') {
        if (!isExactVersion(pkg.version)) {
          errors.push('package.json version must be an exact semver');
        } else {
          version = pkg.version;
        }
      }
      if (typeof pkg.publishConfig?.registry === 'string' && pkg.publishConfig.registry.trim() !== '') {
        publishRegistry = pkg.publishConfig.registry.trim();
      }
      if (typeof pkg.skill?.sourceType === 'string' && pkg.skill.sourceType.trim() !== '') {
        sourceType = pkg.skill.sourceType.trim();
      }
    } catch {
      errors.push('package.json must contain valid JSON');
    }
  }

  return {
    directory,
    skillMdPath,
    packageJsonPath,
    displayName,
    description,
    packageName,
    version,
    publishRegistry,
    sourceType,
    errors,
  };
}
