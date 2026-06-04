import { spawnSync } from 'child_process';
import { mkdtempSync, mkdirSync, rmSync } from 'fs';
import { lstat, readdir, copyFile } from 'fs/promises';
import { tmpdir } from 'os';
import { dirname, join, resolve } from 'path';
import { computeManifestDigest } from './authoring.ts';

const IMPORT_REPORT_PATH = '.skill-packaging-import-report.json';
const PACKAGING_IGNORED_DIRECTORIES = new Set(['.dist', '.git']);
const CAPABILITY_ROOTS = new Set(['commands', 'agents', 'hooks', 'scripts', 'docs']);
const CAPABILITY_FILES = new Set(['plugin.json', '.mcp.json']);

function isIgnoredMacMetadataName(name: string): boolean {
  return name === '.DS_Store' || name.startsWith('._');
}

export interface LocalPackSkillDirectoryOptions {
  directory: string;
}

export interface LocalPackSkillDirectoryResult {
  bundlePath: string;
  manifestDigest: string;
  cleanup: () => void;
}

function shouldIgnorePackagingPath(relativePath: string): boolean {
  const cleanPath = relativePath.replace(/\\/g, '/').trim().replace(/^\.\/+/, '');
  if (!cleanPath || cleanPath === '.') {
    return false;
  }
  if (cleanPath === IMPORT_REPORT_PATH) {
    return true;
  }
  return cleanPath
    .split('/')
    .some((segment) => PACKAGING_IGNORED_DIRECTORIES.has(segment) || isIgnoredMacMetadataName(segment));
}

export async function validateLocalWorkspace(directory: string): Promise<string[]> {
  const lowercasePaths = new Map<string, string>();
  let nestedSkill = false;
  let hasCapability = false;
  let unsafePathEntry = false;

  async function walk(currentDir: string, relativePrefix = ''): Promise<void> {
    const entries = await readdir(currentDir, { withFileTypes: true });

    for (const entry of entries) {
      const relativePath = relativePrefix ? `${relativePrefix}/${entry.name}` : entry.name;
      if (shouldIgnorePackagingPath(relativePath)) {
        continue;
      }

      const stats = await lstat(`${currentDir}/${entry.name}`);
      if (stats.isSymbolicLink()) {
        unsafePathEntry = true;
        continue;
      }

      const lowercasePath = relativePath.toLowerCase();
      const existing = lowercasePaths.get(lowercasePath);
      if (existing && existing !== relativePath) {
        unsafePathEntry = true;
      } else {
        lowercasePaths.set(lowercasePath, relativePath);
      }

      if (entry.isDirectory()) {
        if (CAPABILITY_ROOTS.has(relativePath.split('/')[0] || '')) {
          hasCapability = true;
        }
        await walk(`${currentDir}/${entry.name}`, relativePath);
        continue;
      }

      if (relativePath === 'SKILL.md') {
        hasCapability = true;
      }
      if (relativePath.endsWith('/SKILL.md')) {
        nestedSkill = true;
      }
      if (CAPABILITY_FILES.has(relativePath)) {
        hasCapability = true;
      }
    }
  }

  await walk(directory);

  const errors: string[] = [];
  if (unsafePathEntry) {
    errors.push('unsafe_path_entry');
  }
  if (nestedSkill) {
    errors.push('nested_skill_md');
  }
  if (!hasCapability) {
    errors.push('missing_capability_content');
  }
  return errors;
}

async function copyPublishableWorkspace(sourceDirectory: string, targetDirectory: string): Promise<void> {
  async function walk(currentSourceDir: string, relativePrefix = ''): Promise<void> {
    const entries = await readdir(currentSourceDir, { withFileTypes: true });

    for (const entry of entries) {
      const relativePath = relativePrefix ? `${relativePrefix}/${entry.name}` : entry.name;
      if (shouldIgnorePackagingPath(relativePath)) {
        continue;
      }

      const sourcePath = join(currentSourceDir, entry.name);
      const targetPath = join(targetDirectory, relativePath);
      const stats = await lstat(sourcePath);
      if (stats.isSymbolicLink()) {
        throw new Error('unsafe_path_entry');
      }

      if (entry.isDirectory()) {
        mkdirSync(targetPath, { recursive: true });
        await walk(sourcePath, relativePath);
        continue;
      }

      mkdirSync(dirname(targetPath), { recursive: true });
      await copyFile(sourcePath, targetPath);
    }
  }

  await walk(sourceDirectory);
}

function packWorkspace(workspacePath: string): string {
  const result = spawnSync('npm', ['pack'], {
    cwd: workspacePath,
    stdio: ['inherit', 'pipe', 'pipe'],
    encoding: 'utf-8',
    shell: process.platform === 'win32',
  });

  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || 'npm pack failed');
  }

  const bundleName = (result.stdout || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .at(-1);
  if (!bundleName) {
    throw new Error('npm pack did not report a bundle filename');
  }

  return join(workspacePath, bundleName);
}

export async function packLocalSkillDirectory(
  options: LocalPackSkillDirectoryOptions
): Promise<LocalPackSkillDirectoryResult> {
  const directory = resolve(options.directory);
  const tempDir = mkdtempSync(join(tmpdir(), 'skills-publish-'));
  const workspacePath = join(tempDir, 'package');
  mkdirSync(workspacePath, { recursive: true });

  try {
    await copyPublishableWorkspace(directory, workspacePath);
    const bundlePath = packWorkspace(workspacePath);
    const manifestDigest = await computeManifestDigest(directory);

    return {
      bundlePath,
      manifestDigest,
      cleanup: () => rmSync(tempDir, { recursive: true, force: true }),
    };
  } catch (error) {
    rmSync(tempDir, { recursive: true, force: true });
    throw error;
  }
}
