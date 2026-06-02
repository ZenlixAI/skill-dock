import { readFile, writeFile, readdir, stat } from 'fs/promises';
import { join, relative } from 'path';
import { createHash } from 'crypto';

const LOCAL_LOCK_FILE = 'skills-lock.json';
const CURRENT_VERSION = 1;

export interface LocalSkillLegacyMetadata {
  /** Historical source identifier from pre-registry install flows. */
  source?: string;
  /** Historical branch or tag ref from pre-registry install flows. */
  ref?: string;
  /** Historical repo-local SKILL.md path from pre-registry install flows. */
  skillPath?: string;
}

/**
 * Represents a single skill entry in the local (project) lock file.
 *
 * Intentionally minimal and timestamp-free to minimize merge conflicts.
 * Two branches adding different skills produce non-overlapping JSON keys
 * that git can auto-merge cleanly.
 */
export interface LocalSkillLockEntry {
  /** Registry identity for registry-installed skills. */
  packageName?: string;
  /** Resolved installed version for registry-installed skills. */
  version?: string;
  /** Catalog base URL used for resolution. */
  registry?: string;
  /** Integrity string returned by the install-target endpoint. */
  integrity?: string;
  /** Catalog-generated manifest digest for the installed bundle. */
  skillManifestDigest?: string;
  /**
   * Install source classification.
   *
   * `registry` is the canonical V1 path. Other values such as `node_modules`
   * or `local` exist only so the lock reader can classify historical entries
   * without treating them as the primary contract.
   */
  sourceType: string;
  /** Historical compatibility-only metadata from pre-registry install flows. */
  legacy?: LocalSkillLegacyMetadata;
  /**
   * SHA-256 hash computed from all files in the skill folder.
   * Unlike the global lock which uses GitHub tree SHA, the local lock
   * computes the hash from actual file contents on disk.
   */
  computedHash: string;
}

/**
 * The structure of the local (project-scoped) skill lock file.
 * This file is meant to be checked into version control.
 *
 * Skills are sorted alphabetically by name when written to produce
 * deterministic output and minimize merge conflicts.
 */
export interface LocalSkillLockFile {
  /** Schema version for future migrations */
  version: number;
  /** Map of skill name to its lock entry (sorted alphabetically). */
  skills: Record<string, LocalSkillLockEntry>;
}

/**
 * Get the path to the local skill lock file for a project.
 */
export function getLocalLockPath(cwd?: string): string {
  return join(cwd || process.cwd(), LOCAL_LOCK_FILE);
}

/**
 * Read the local skill lock file.
 * Returns an empty lock file structure if the file doesn't exist
 * or is corrupted (e.g., merge conflict markers).
 */
export async function readLocalLock(cwd?: string): Promise<LocalSkillLockFile> {
  const lockPath = getLocalLockPath(cwd);

  try {
    const content = await readFile(lockPath, 'utf-8');
    const parsed = JSON.parse(content) as LocalSkillLockFile;

    if (typeof parsed.version !== 'number' || !parsed.skills) {
      return createEmptyLocalLock();
    }

    if (parsed.version < CURRENT_VERSION) {
      return migrateLocalLock(parsed);
    }

    return normalizeLocalLock(parsed);
  } catch {
    return createEmptyLocalLock();
  }
}

function migrateLocalLock(parsed: LocalSkillLockFile): LocalSkillLockFile {
  return normalizeLocalLock({
    version: CURRENT_VERSION,
    skills: parsed.skills || {},
  });
}

function normalizeLocalLock(parsed: LocalSkillLockFile): LocalSkillLockFile {
  const migratedSkills: Record<string, LocalSkillLockEntry> = {};

  for (const [name, rawEntry] of Object.entries(parsed.skills || {})) {
    migratedSkills[name] = normalizeLocalSkillLockEntry(rawEntry);
  }

  return {
    version: CURRENT_VERSION,
    skills: migratedSkills,
  };
}

/**
 * Write the local skill lock file.
 * Skills are sorted alphabetically by name for deterministic output.
 */
export async function writeLocalLock(lock: LocalSkillLockFile, cwd?: string): Promise<void> {
  const lockPath = getLocalLockPath(cwd);

  // Sort skills alphabetically for deterministic output / clean diffs
  const sortedSkills: Record<string, Record<string, unknown>> = {};
  for (const key of Object.keys(lock.skills).sort()) {
    sortedSkills[key] = serializeLocalSkillLockEntry(lock.skills[key]!);
  }

  const sorted = { version: lock.version, skills: sortedSkills };
  const content = JSON.stringify(sorted, null, 2) + '\n';
  await writeFile(lockPath, content, 'utf-8');
}

/**
 * Compute a SHA-256 hash from all files in a skill directory.
 * Reads all files recursively, sorts them by relative path for determinism,
 * and produces a single hash from their concatenated contents.
 */
export async function computeSkillFolderHash(skillDir: string): Promise<string> {
  const files: Array<{ relativePath: string; content: Buffer }> = [];
  await collectFiles(skillDir, skillDir, files);

  // Sort by relative path for deterministic hashing
  files.sort((a, b) => a.relativePath.localeCompare(b.relativePath));

  const hash = createHash('sha256');
  for (const file of files) {
    // Include the path in the hash so renames are detected
    hash.update(file.relativePath);
    hash.update(file.content);
  }

  return hash.digest('hex');
}

async function collectFiles(
  baseDir: string,
  currentDir: string,
  results: Array<{ relativePath: string; content: Buffer }>
): Promise<void> {
  const entries = await readdir(currentDir, { withFileTypes: true });

  await Promise.all(
    entries.map(async (entry) => {
      const fullPath = join(currentDir, entry.name);

      if (entry.isDirectory()) {
        // Skip .git and node_modules within skill dirs
        if (entry.name === '.git' || entry.name === 'node_modules') return;
        await collectFiles(baseDir, fullPath, results);
      } else if (entry.isFile()) {
        const content = await readFile(fullPath);
        const relativePath = relative(baseDir, fullPath).split('\\').join('/');
        results.push({ relativePath, content });
      }
    })
  );
}

/**
 * Add or update a skill entry in the local lock file.
 */
export async function addSkillToLocalLock(
  skillName: string,
  entry: LocalSkillLockEntry,
  cwd?: string
): Promise<void> {
  const lock = await readLocalLock(cwd);
  lock.skills[skillName] = entry;
  await writeLocalLock(lock, cwd);
}

/**
 * Remove a skill from the local lock file.
 */
export async function removeSkillFromLocalLock(skillName: string, cwd?: string): Promise<boolean> {
  const lock = await readLocalLock(cwd);

  if (!(skillName in lock.skills)) {
    return false;
  }

  delete lock.skills[skillName];
  await writeLocalLock(lock, cwd);
  return true;
}

function createEmptyLocalLock(): LocalSkillLockFile {
  return {
    version: CURRENT_VERSION,
    skills: {},
  };
}

type RawLocalSkillLockEntry = LocalSkillLockEntry & {
  source?: string;
  ref?: string;
  skillPath?: string;
};

function normalizeLocalSkillLockEntry(rawEntry: unknown): LocalSkillLockEntry {
  const entry = (rawEntry || {}) as Partial<RawLocalSkillLockEntry>;
  const legacy = normalizeLocalLegacyMetadata(entry);

  return {
    sourceType: entry.sourceType || 'registry',
    computedHash: entry.computedHash || '',
    packageName: entry.packageName,
    version: entry.version,
    registry: entry.registry,
    integrity: entry.integrity,
    skillManifestDigest: entry.skillManifestDigest,
    ...(legacy ? { legacy } : {}),
  };
}

function normalizeLocalLegacyMetadata(
  entry: Partial<RawLocalSkillLockEntry>
): LocalSkillLegacyMetadata | undefined {
  const source = entry.legacy?.source ?? entry.source;
  const ref = entry.legacy?.ref ?? entry.ref;
  const skillPath = entry.legacy?.skillPath ?? entry.skillPath;

  if (!source && !ref && !skillPath) {
    return undefined;
  }

  return {
    ...(source ? { source } : {}),
    ...(ref ? { ref } : {}),
    ...(skillPath ? { skillPath } : {}),
  };
}

function serializeLocalSkillLockEntry(entry: LocalSkillLockEntry): Record<string, unknown> {
  return {
    ...(entry.legacy?.source ? { source: entry.legacy.source } : {}),
    ...(entry.legacy?.ref ? { ref: entry.legacy.ref } : {}),
    ...(entry.legacy?.skillPath ? { skillPath: entry.legacy.skillPath } : {}),
    sourceType: entry.sourceType,
    computedHash: entry.computedHash,
    ...(entry.packageName ? { packageName: entry.packageName } : {}),
    ...(entry.version ? { version: entry.version } : {}),
    ...(entry.registry ? { registry: entry.registry } : {}),
    ...(entry.integrity ? { integrity: entry.integrity } : {}),
    ...(entry.skillManifestDigest ? { skillManifestDigest: entry.skillManifestDigest } : {}),
  };
}
