/**
 * GitHub tree inspection utilities used by lock maintenance.
 */

// ─── Constants ───

/** Timeout for individual HTTP fetches (ms) */
const FETCH_TIMEOUT = 10_000;

// ─── GitHub Trees API ───

export interface TreeEntry {
  path: string;
  type: 'blob' | 'tree';
  sha: string;
  size?: number;
}

export interface RepoTree {
  sha: string;
  branch: string;
  tree: TreeEntry[];
}

/**
 * Within-process memo: once we've discovered that the GitHub API has
 * rate-limited this IP, subsequent calls skip straight to the auth fallback
 * instead of burning round trips on requests guaranteed to 403.
 */
let _rateLimitedThisSession = false;

/** For tests only. */
export function resetRepoTreeAuthState(): void {
  _rateLimitedThisSession = false;
}

interface BranchFetchResult {
  tree: RepoTree | null;
  rateLimited: boolean;
}

async function fetchTreeBranch(
  ownerRepo: string,
  branch: string,
  token: string | null
): Promise<BranchFetchResult> {
  try {
    const url = `https://api.github.com/repos/${ownerRepo}/git/trees/${encodeURIComponent(branch)}?recursive=1`;
    const headers: Record<string, string> = {
      Accept: 'application/vnd.github.v3+json',
      'User-Agent': 'skills-cli',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      headers,
      signal: AbortSignal.timeout(FETCH_TIMEOUT),
    });

    if (response.ok) {
      const data = (await response.json()) as {
        sha: string;
        tree: TreeEntry[];
      };
      return {
        tree: { sha: data.sha, branch, tree: data.tree },
        rateLimited: false,
      };
    }

    // GitHub signals rate-limit with 403 + X-RateLimit-Remaining: 0.
    // (A bare 403 means permission denied, which is not retryable here.)
    const rateLimited =
      response.status === 403 && response.headers.get('x-ratelimit-remaining') === '0';
    return { tree: null, rateLimited };
  } catch {
    return { tree: null, rateLimited: false };
  }
}

/**
 * Fetch the full recursive tree for a GitHub repo.
 * Returns the tree data including all entries, or null on failure.
 * Tries branches in order: ref (if specified), then main, then master.
 *
 * Authentication is lazy: by default the call goes out unauthenticated,
 * which is enough for the vast majority of users (60 req/hr per IP).
 * Only if GitHub responds with a rate-limit 403 do we ask the optional
 * `getToken` callback for a token and retry. This avoids invoking
 * `gh auth token` on every install, which corporate endpoint security
 * tools flag as suspicious credential extraction. See issue #523.
 */
export async function fetchRepoTree(
  ownerRepo: string,
  ref?: string,
  getToken?: () => string | null
): Promise<RepoTree | null> {
  const branches = ref ? [ref] : ['HEAD', 'main', 'master'];

  // Fast path: once we've seen a rate limit in this process, don't bother
  // retrying unauth on subsequent calls. Go straight to auth.
  if (_rateLimitedThisSession && getToken) {
    const token = getToken();
    if (!token) return null;
    for (const branch of branches) {
      const result = await fetchTreeBranch(ownerRepo, branch, token);
      if (result.tree) return result.tree;
    }
    return null;
  }

  // First pass: unauthenticated.
  let rateLimited = false;
  for (const branch of branches) {
    const result = await fetchTreeBranch(ownerRepo, branch, null);
    if (result.tree) return result.tree;
    if (result.rateLimited) {
      // All branches share the same rate-limit bucket on this IP, so it's
      // pointless to keep trying other branches in this pass.
      rateLimited = true;
      break;
    }
  }

  if (!rateLimited || !getToken) return null;

  // Lazy fallback: rate limit hit and a token resolver was provided.
  _rateLimitedThisSession = true;
  const token = getToken();
  if (!token) return null;

  for (const branch of branches) {
    const result = await fetchTreeBranch(ownerRepo, branch, token);
    if (result.tree) return result.tree;
  }
  return null;
}

/**
 * Extract the folder hash (tree SHA) for a specific skill path from a repo tree.
 * This replaces the per-skill GitHub API call previously done in fetchSkillFolderHash().
 */
export function getSkillFolderHashFromTree(tree: RepoTree, skillPath: string): string | null {
  let folderPath = skillPath.replace(/\\/g, '/');

  // Remove SKILL.md suffix to get folder path (case-insensitive)
  if (folderPath.toLowerCase().endsWith('/skill.md')) {
    folderPath = folderPath.slice(0, -9);
  } else if (folderPath.toLowerCase().endsWith('skill.md')) {
    folderPath = folderPath.slice(0, -8);
  }
  if (folderPath.endsWith('/')) {
    folderPath = folderPath.slice(0, -1);
  }

  // Root-level skill
  if (!folderPath) {
    return tree.sha;
  }

  const entry = tree.tree.find((e) => e.type === 'tree' && e.path === folderPath);
  return entry?.sha ?? null;
}

// ─── Skill discovery from tree ───

/** Known directories where SKILL.md files are commonly found (relative to repo root) */
const PRIORITY_PREFIXES = [
  '',
  'skills/',
  'skills/.curated/',
  'skills/.experimental/',
  'skills/.system/',
  '.agents/skills/',
  '.claude/skills/',
  '.cline/skills/',
  '.codebuddy/skills/',
  '.codex/skills/',
  '.commandcode/skills/',
  '.continue/skills/',
  '.github/skills/',
  '.goose/skills/',
  '.iflow/skills/',
  '.junie/skills/',
  '.kilocode/skills/',
  '.kiro/skills/',
  '.mux/skills/',
  '.neovate/skills/',
  '.opencode/skills/',
  '.openhands/skills/',
  '.pi/skills/',
  '.qoder/skills/',
  '.roo/skills/',
  '.trae/skills/',
  '.windsurf/skills/',
  '.zencoder/skills/',
];

/**
 * Find all SKILL.md file paths in a repo tree.
 * Applies the same priority directory logic as discoverSkills().
 * If subpath is set, only searches within that subtree.
 */
export function findSkillMdPaths(tree: RepoTree, subpath?: string): string[] {
  // Find all blob entries that are SKILL.md files (case-insensitive)
  const allSkillMds = tree.tree
    .filter((e) => e.type === 'blob' && e.path.toLowerCase().endsWith('skill.md'))
    .map((e) => e.path);

  // Apply subpath filter
  const prefix = subpath ? (subpath.endsWith('/') ? subpath : subpath + '/') : '';
  const filtered = prefix
    ? allSkillMds.filter((p) => p.startsWith(prefix) || p === prefix + 'SKILL.md')
    : allSkillMds;

  if (filtered.length === 0) return [];

  // Check priority directories first (same order as discoverSkills).
  // Non-root prefixes also accept depth-2 paths so the blob fast path stays
  // in sync with the on-disk walk's catalog-layout discovery.
  const priorityResults: string[] = [];
  const seen = new Set<string>();
  // Mirror of SKIP_DIRS at the top of src/skills.ts. Kept local to avoid
  // a cross-file import; if these ever drift, update both.
  const SKIP_DIRS = new Set(['node_modules', '.git', 'dist', 'build', '__pycache__']);
  const lowerSkillMdSet = new Set(filtered.map((p) => p.toLowerCase()));

  for (const priorityPrefix of PRIORITY_PREFIXES) {
    const fullPrefix = prefix + priorityPrefix;
    const isContainer = priorityPrefix !== '';

    for (const skillMd of filtered) {
      // Check if this SKILL.md is directly inside the priority dir (one level deep)
      if (!skillMd.startsWith(fullPrefix)) continue;
      const rest = skillMd.slice(fullPrefix.length);

      // Direct SKILL.md in the priority dir (e.g., "skills/SKILL.md")
      if (rest.toLowerCase() === 'skill.md') {
        if (!seen.has(skillMd)) {
          priorityResults.push(skillMd);
          seen.add(skillMd);
        }
        continue;
      }

      // SKILL.md one level deep (e.g., "skills/react-best-practices/SKILL.md")
      const parts = rest.split('/');
      if (parts.length === 2 && parts[1]!.toLowerCase() === 'skill.md') {
        if (!seen.has(skillMd)) {
          priorityResults.push(skillMd);
          seen.add(skillMd);
        }
        continue;
      }

      // SKILL.md two levels deep under a known container prefix
      // (e.g., "skills/<category>/<skill>/SKILL.md"). Skip if the parent
      // child dir already has its own SKILL.md (no descent past), or if
      // any path segment is an ignored directory.
      if (
        isContainer &&
        parts.length === 3 &&
        parts[2]!.toLowerCase() === 'skill.md' &&
        !SKIP_DIRS.has(parts[0]!) &&
        !SKIP_DIRS.has(parts[1]!)
      ) {
        const parentSkillMd = `${fullPrefix}${parts[0]}/SKILL.md`.toLowerCase();
        if (!lowerSkillMdSet.has(parentSkillMd) && !seen.has(skillMd)) {
          priorityResults.push(skillMd);
          seen.add(skillMd);
        }
      }
    }
  }

  // If we found skills in priority dirs, return those
  if (priorityResults.length > 0) return priorityResults;

  // Fallback: return all SKILL.md files found (limited to 5 levels deep)
  return filtered.filter((p) => {
    const depth = p.split('/').length;
    return depth <= 6; // 5 levels + the SKILL.md file itself
  });
}
