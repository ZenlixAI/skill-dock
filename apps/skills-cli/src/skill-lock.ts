import { readFile, writeFile, mkdir } from 'fs/promises';
import { join, dirname } from 'path';
import { homedir } from 'os';
import { createHash } from 'crypto';
import { execSync } from 'child_process';
import pc from 'picocolors';

const AGENTS_DIR = '.agents';
const LOCK_FILE = '.skill-lock.json';
const CURRENT_VERSION = 3; // Bumped from 2 to 3 for folder hash support (GitHub tree SHA)

export interface SkillLockLegacyMetadata {
  /** Historical source identifier from pre-registry install flows. */
  source?: string;
  /** Historical fetch URL from pre-registry install flows. */
  sourceUrl?: string;
  /** Historical branch or tag ref from pre-registry install flows. */
  ref?: string;
  /** Historical repo-local SKILL.md path from pre-registry install flows. */
  skillPath?: string;
}

/**
 * Represents a single installed skill entry in the lock file.
 */
export interface SkillLockEntry {
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
   * `registry` is the intended V1 steady-state. Other values remain so the CLI
   * can classify historical global installs without making them first-class.
   */
  sourceType: string;
  /** Historical compatibility-only metadata from pre-registry install flows. */
  legacy?: SkillLockLegacyMetadata;
  /**
   * GitHub tree SHA for the entire skill folder.
   * This hash changes when ANY file in the skill folder changes.
   * Fetched via GitHub Trees API by the telemetry server.
   */
  skillFolderHash: string;
  /** ISO timestamp when the skill was first installed */
  installedAt: string;
  /** ISO timestamp when the skill was last updated */
  updatedAt: string;
  /** Name of the plugin this skill belongs to (if any) */
  pluginName?: string;
}

/**
 * Tracks dismissed prompts so they're not shown again.
 */
export interface DismissedPrompts {
  /** Dismissed the find-skills skill installation prompt */
  findSkillsPrompt?: boolean;
}

/**
 * The structure of the skill lock file.
 */
export interface SkillLockFile {
  /** Schema version for future migrations */
  version: number;
  /** Map of skill name to its lock entry */
  skills: Record<string, SkillLockEntry>;
  /** Tracks dismissed prompts */
  dismissed?: DismissedPrompts;
  /** Last selected agents for installation */
  lastSelectedAgents?: string[];
}

/**
 * Get the path to the global skill lock file.
 * Use $XDG_STATE_HOME/skills/.skill-lock.json if set.
 * otherwise fall back to ~/.agents/.skill-lock.json
 */
export function getSkillLockPath(): string {
  const xdgStateHome = process.env.XDG_STATE_HOME;
  if (xdgStateHome) {
    return join(xdgStateHome, 'skills', LOCK_FILE);
  }
  return join(homedir(), AGENTS_DIR, LOCK_FILE);
}

/**
 * Read the skill lock file.
 * Returns an empty lock file structure if the file doesn't exist.
 * Wipes the lock file if it's an old format (version < CURRENT_VERSION).
 */
export async function readSkillLock(): Promise<SkillLockFile> {
  const lockPath = getSkillLockPath();

  try {
    const content = await readFile(lockPath, 'utf-8');
    const parsed = JSON.parse(content) as SkillLockFile;

    // Validate version - wipe if old format
    if (typeof parsed.version !== 'number' || !parsed.skills) {
      return createEmptyLockFile();
    }

    if (parsed.version < CURRENT_VERSION) {
      return migrateSkillLock(parsed);
    }

    return normalizeSkillLock(parsed);
  } catch (error) {
    // File doesn't exist or is invalid - return empty
    return createEmptyLockFile();
  }
}

function migrateSkillLock(parsed: SkillLockFile): SkillLockFile {
  return normalizeSkillLock({
    version: CURRENT_VERSION,
    skills: parsed.skills || {},
    dismissed: parsed.dismissed || {},
    lastSelectedAgents: parsed.lastSelectedAgents,
  });
}

function normalizeSkillLock(parsed: SkillLockFile): SkillLockFile {
  const migratedSkills: Record<string, SkillLockEntry> = {};

  for (const [name, rawEntry] of Object.entries(parsed.skills || {})) {
    migratedSkills[name] = normalizeSkillLockEntry(rawEntry);
  }

  return {
    version: CURRENT_VERSION,
    skills: migratedSkills,
    dismissed: parsed.dismissed || {},
    lastSelectedAgents: parsed.lastSelectedAgents,
  };
}

/**
 * Write the skill lock file.
 * Creates the directory if it doesn't exist.
 */
export async function writeSkillLock(lock: SkillLockFile): Promise<void> {
  const lockPath = getSkillLockPath();

  // Ensure directory exists
  await mkdir(dirname(lockPath), { recursive: true });

  // Write with pretty formatting for human readability
  const serializedSkills = Object.fromEntries(
    Object.entries(lock.skills).map(([name, entry]) => [name, serializeSkillLockEntry(entry)])
  );
  const content = JSON.stringify({ ...lock, skills: serializedSkills }, null, 2);
  await writeFile(lockPath, content, 'utf-8');
}

/**
 * Compute SHA-256 hash of content.
 */
export function computeContentHash(content: string): string {
  return createHash('sha256').update(content, 'utf-8').digest('hex');
}

let _ghWarningShown = false;

/** For tests only. Resets the one-shot warning flag. */
export function resetGhAuthWarning(): void {
  _ghWarningShown = false;
}

/**
 * Get GitHub token from user's environment.
 * Tries in order:
 * 1. GITHUB_TOKEN environment variable (silent)
 * 2. GH_TOKEN environment variable (silent)
 * 3. gh CLI auth token, if gh is installed. Prints a one-time warning to
 *    stderr before invoking `gh auth token`, because that subprocess call
 *    is flagged by some corporate endpoint security tooling (Defender, etc.)
 *    as credential extraction. Callers should invoke this function lazily
 *    (e.g. only after an unauthenticated request hits a rate limit) so the
 *    fallback rarely runs in practice.
 *
 * @returns The token string or null if not available
 */
export function getGitHubToken(): string | null {
  // Check environment variables first (silent: user has explicitly opted in)
  if (process.env.GITHUB_TOKEN) {
    return process.env.GITHUB_TOKEN;
  }
  if (process.env.GH_TOKEN) {
    return process.env.GH_TOKEN;
  }

  // Last resort: spawn gh CLI. Warn the user once per process before doing so.
  if (!_ghWarningShown) {
    process.stderr.write(
      `${pc.yellow('│')}  ${pc.yellow('GitHub rate limit reached')} — using your ${pc.cyan('gh')} login to continue.\n` +
        `${pc.yellow('│')}  ${pc.dim(`Tip: set ${pc.cyan('GITHUB_TOKEN')} to avoid this prompt, or use ${pc.cyan('--full-depth')} to clone instead.\n`)}`
    );
    _ghWarningShown = true;
  }
  try {
    const token = execSync('gh auth token', {
      encoding: 'utf-8',
      stdio: ['pipe', 'pipe', 'pipe'],
    }).trim();
    if (token) {
      return token;
    }
  } catch {
    // gh not installed or not authenticated
  }

  return null;
}

/**
 * Fetch the tree SHA (folder hash) for a skill folder using GitHub's Trees API.
 * This makes ONE API call to get the entire repo tree, then extracts the SHA
 * for the specific skill folder.
 *
 * @param ownerRepo - GitHub owner/repo (e.g., "vercel-labs/agent-skills")
 * @param skillPath - Path to skill folder or SKILL.md (e.g., "skills/react-best-practices/SKILL.md")
 * @param getToken - Optional lazy token resolver. Invoked only if the
 *                   unauthenticated request hits a rate limit.
 * @param ref - Optional branch/tag ref. Defaults to trying main then master.
 * @returns The tree SHA for the skill folder, or null if not found
 */
export async function fetchSkillFolderHash(
  ownerRepo: string,
  skillPath: string,
  getToken?: (() => string | null) | null,
  ref?: string
): Promise<string | null> {
  const { fetchRepoTree, getSkillFolderHashFromTree } = await import('./blob.ts');
  const tree = await fetchRepoTree(ownerRepo, ref, getToken ?? undefined);
  if (!tree) return null;
  return getSkillFolderHashFromTree(tree, skillPath);
}

/**
 * Add or update a skill entry in the lock file.
 */
export async function addSkillToLock(
  skillName: string,
  entry: Omit<SkillLockEntry, 'installedAt' | 'updatedAt'>
): Promise<void> {
  const lock = await readSkillLock();
  const now = new Date().toISOString();

  const existingEntry = lock.skills[skillName];

  lock.skills[skillName] = {
    ...entry,
    installedAt: existingEntry?.installedAt ?? now,
    updatedAt: now,
  };

  await writeSkillLock(lock);
}

/**
 * Remove a skill from the lock file.
 */
export async function removeSkillFromLock(skillName: string): Promise<boolean> {
  const lock = await readSkillLock();

  if (!(skillName in lock.skills)) {
    return false;
  }

  delete lock.skills[skillName];
  await writeSkillLock(lock);
  return true;
}

/**
 * Get a skill entry from the lock file.
 */
export async function getSkillFromLock(skillName: string): Promise<SkillLockEntry | null> {
  const lock = await readSkillLock();
  return lock.skills[skillName] ?? null;
}

/**
 * Get all skills from the lock file.
 */
export async function getAllLockedSkills(): Promise<Record<string, SkillLockEntry>> {
  const lock = await readSkillLock();
  return lock.skills;
}

/**
 * Get skills grouped by source for batch update operations.
 */
export async function getSkillsBySource(): Promise<
  Map<string, { skills: string[]; entry: SkillLockEntry }>
> {
  const lock = await readSkillLock();
  const bySource = new Map<string, { skills: string[]; entry: SkillLockEntry }>();

  for (const [skillName, entry] of Object.entries(lock.skills)) {
    const sourceKey = entry.packageName || entry.legacy?.source || skillName;
    const existing = bySource.get(sourceKey);
    if (existing) {
      existing.skills.push(skillName);
    } else {
      bySource.set(sourceKey, { skills: [skillName], entry });
    }
  }

  return bySource;
}

/**
 * Create an empty lock file structure.
 */
function createEmptyLockFile(): SkillLockFile {
  return {
    version: CURRENT_VERSION,
    skills: {},
    dismissed: {},
  };
}

type RawSkillLockEntry = SkillLockEntry & {
  source?: string;
  sourceUrl?: string;
  ref?: string;
  skillPath?: string;
};

function normalizeSkillLockEntry(rawEntry: unknown): SkillLockEntry {
  const entry = (rawEntry || {}) as Partial<RawSkillLockEntry>;
  const legacy = normalizeSkillLegacyMetadata(entry);

  return {
    sourceType: entry.sourceType || 'registry',
    skillFolderHash: entry.skillFolderHash || '',
    installedAt: entry.installedAt || '',
    updatedAt: entry.updatedAt || '',
    packageName: entry.packageName,
    version: entry.version,
    registry: entry.registry,
    integrity: entry.integrity,
    skillManifestDigest: entry.skillManifestDigest,
    pluginName: entry.pluginName,
    ...(legacy ? { legacy } : {}),
  };
}

function normalizeSkillLegacyMetadata(
  entry: Partial<RawSkillLockEntry>
): SkillLockLegacyMetadata | undefined {
  const source = entry.legacy?.source ?? entry.source;
  const sourceUrl = entry.legacy?.sourceUrl ?? entry.sourceUrl;
  const ref = entry.legacy?.ref ?? entry.ref;
  const skillPath = entry.legacy?.skillPath ?? entry.skillPath;

  if (!source && !sourceUrl && !ref && !skillPath) {
    return undefined;
  }

  return {
    ...(source ? { source } : {}),
    ...(sourceUrl ? { sourceUrl } : {}),
    ...(ref ? { ref } : {}),
    ...(skillPath ? { skillPath } : {}),
  };
}

function serializeSkillLockEntry(entry: SkillLockEntry): Record<string, unknown> {
  return {
    ...(entry.legacy?.source ? { source: entry.legacy.source } : {}),
    ...(entry.legacy?.sourceUrl ? { sourceUrl: entry.legacy.sourceUrl } : {}),
    ...(entry.legacy?.ref ? { ref: entry.legacy.ref } : {}),
    ...(entry.legacy?.skillPath ? { skillPath: entry.legacy.skillPath } : {}),
    sourceType: entry.sourceType,
    skillFolderHash: entry.skillFolderHash,
    installedAt: entry.installedAt,
    updatedAt: entry.updatedAt,
    ...(entry.packageName ? { packageName: entry.packageName } : {}),
    ...(entry.version ? { version: entry.version } : {}),
    ...(entry.registry ? { registry: entry.registry } : {}),
    ...(entry.integrity ? { integrity: entry.integrity } : {}),
    ...(entry.skillManifestDigest ? { skillManifestDigest: entry.skillManifestDigest } : {}),
    ...(entry.pluginName ? { pluginName: entry.pluginName } : {}),
  };
}

/**
 * Check if a prompt has been dismissed.
 */
export async function isPromptDismissed(promptKey: keyof DismissedPrompts): Promise<boolean> {
  const lock = await readSkillLock();
  return lock.dismissed?.[promptKey] === true;
}

/**
 * Mark a prompt as dismissed.
 */
export async function dismissPrompt(promptKey: keyof DismissedPrompts): Promise<void> {
  const lock = await readSkillLock();
  if (!lock.dismissed) {
    lock.dismissed = {};
  }
  lock.dismissed[promptKey] = true;
  await writeSkillLock(lock);
}

/**
 * Get the last selected agents.
 */
export async function getLastSelectedAgents(): Promise<string[] | undefined> {
  const lock = await readSkillLock();
  return lock.lastSelectedAgents;
}

/**
 * Save the selected agents to the lock file.
 */
export async function saveSelectedAgents(agents: string[]): Promise<void> {
  const lock = await readSkillLock();
  lock.lastSelectedAgents = agents;
  await writeSkillLock(lock);
}
