import { existsSync } from 'fs';
import { resolve } from 'path';
import type { ParsedSource } from './types.ts';

const GITHUB_SHORTHAND_RE =
  /^(?<owner>[A-Za-z0-9_.-]+)\/(?<repo>[A-Za-z0-9_.-]+?)(?:@(?<skill>[^/?#]+))?$/;
const GITLAB_HOST_PREFIX = 'gitlab.';
const GITLAB_TREE_RE = /^(?<repo>.+?)\/-\/tree\/(?<ref>[^/]+)(?:\/(?<subpath>.+?))?\/?$/;

function normalizeSourceSubpath(raw: string | null | undefined): string | undefined {
  if (!raw) return undefined;
  const normalized = raw.trim().replace(/^\/+/, '');
  return normalized.length > 0 ? normalized : undefined;
}

function normalizeRepoPath(raw: string): string {
  return raw
    .trim()
    .replace(/^\/+|\/+$/g, '')
    .replace(/\.git$/i, '');
}

function isGitLabHost(hostname: string): boolean {
  return hostname.toLowerCase().startsWith(GITLAB_HOST_PREFIX);
}

function buildGitLabSshUrl(hostname: string, repoPath: string): string {
  return `git@${hostname}:${normalizeRepoPath(repoPath)}.git`;
}

function splitGitLabSkillFilter(raw: string): { source: string; skillFilter?: string } {
  if (raw.includes('?') || raw.includes('#') || raw.includes('/-/')) {
    return { source: raw };
  }

  const atIndex = raw.lastIndexOf('@');
  if (atIndex <= 0) {
    return { source: raw };
  }

  const skillFilter = raw.slice(atIndex + 1).trim();
  return {
    source: raw.slice(0, atIndex),
    ...(skillFilter ? { skillFilter } : {}),
  };
}

function parseGitLabHttpSource(
  parsed: URL,
  subpath: string | undefined,
  ref: string | undefined
): ParsedSource | null {
  if (!isGitLabHost(parsed.hostname)) {
    return null;
  }

  const pathname = parsed.pathname.replace(/^\/+/, '');
  const treeMatch = pathname.match(GITLAB_TREE_RE);
  if (treeMatch?.groups) {
    const repoPath = normalizeRepoPath(treeMatch.groups.repo ?? '');
    if (!repoPath.includes('/')) {
      return null;
    }

    const treeRef = treeMatch.groups.ref?.trim() || undefined;
    const treeSubpath = normalizeSourceSubpath(treeMatch.groups.subpath);
    return {
      type: 'gitlab',
      url: buildGitLabSshUrl(parsed.hostname, repoPath),
      ...(ref || treeRef ? { ref: ref || treeRef } : {}),
      ...(subpath || treeSubpath ? { subpath: subpath || treeSubpath } : {}),
    };
  }

  // Other GitLab UI paths such as /-/blob/ or /-/issues/ are not install sources.
  if (pathname.includes('/-/')) {
    return null;
  }

  const repoPath = normalizeRepoPath(pathname);
  if (!repoPath.includes('/')) {
    return null;
  }

  return {
    type: 'gitlab',
    url: buildGitLabSshUrl(parsed.hostname, repoPath),
    ...(subpath ? { subpath } : {}),
    ...(ref ? { ref } : {}),
  };
}

function buildGitHttpCloneUrl(parsed: URL, pathname: string): string {
  const credentials = parsed.username
    ? `${parsed.username}${parsed.password ? `:${parsed.password}` : ''}@`
    : '';
  return `${parsed.protocol}//${credentials}${parsed.host}${pathname}`;
}

function parseGitHttpSource(value: string): ParsedSource | null {
  let parsed: URL;
  try {
    parsed = new URL(value);
  } catch {
    return null;
  }

  if (!['http:', 'https:'].includes(parsed.protocol)) {
    return null;
  }

  const subpath = normalizeSourceSubpath(parsed.searchParams.get('path'));
  const ref = parsed.searchParams.get('ref')?.trim() || undefined;

  if (parsed.hostname === 'github.com') {
    const segments = parsed.pathname.split('/').filter(Boolean);
    if (segments.length >= 2) {
      const owner = segments[0]!;
      const repo = segments[1]!.replace(/\.git$/, '');
      return {
        type: 'github',
        url: `https://github.com/${owner}/${repo}.git`,
        ...(subpath ? { subpath } : {}),
        ...(ref ? { ref } : {}),
      };
    }
  }

  const gitlabSource = parseGitLabHttpSource(parsed, subpath, ref);
  if (gitlabSource) {
    return gitlabSource;
  }

  // Unsupported GitLab UI paths should not fall through to generic git parsing.
  if (isGitLabHost(parsed.hostname)) {
    return null;
  }

  const pathname = parsed.pathname.replace(/\/+$/, '');
  if (/\.git$/i.test(pathname)) {
    return {
      type: 'git',
      url: buildGitHttpCloneUrl(parsed, pathname),
      ...(subpath ? { subpath } : {}),
      ...(ref ? { ref } : {}),
    };
  }

  return null;
}

function parseGitLabPrefixSource(value: string): ParsedSource | null {
  const raw = value.slice('gitlab:'.length).trim();
  if (!raw) {
    return null;
  }

  const { source, skillFilter } = splitGitLabSkillFilter(raw);
  const parsed = parseGitHttpSource(`https://gitlab.com/${source.replace(/^\/+/, '')}`);
  if (!parsed || parsed.type !== 'gitlab') {
    return null;
  }

  return {
    ...parsed,
    ...(skillFilter ? { skillFilter } : {}),
  };
}

function parseGitLabHostShorthand(value: string): ParsedSource | null {
  if (!/^gitlab\./i.test(value)) {
    return null;
  }

  const { source, skillFilter } = splitGitLabSkillFilter(value);
  const parsed = parseGitHttpSource(`https://${source}`);
  if (!parsed || parsed.type !== 'gitlab') {
    return null;
  }

  return {
    ...parsed,
    ...(skillFilter ? { skillFilter } : {}),
  };
}

function parseGitSshSource(value: string): ParsedSource | null {
  if (value.startsWith('git@')) {
    const host = value.match(/^git@(?<host>[^:]+):/i)?.groups?.host;
    return {
      type: host && isGitLabHost(host) ? 'gitlab' : 'git',
      url: value,
    };
  }

  if (value.startsWith('ssh://') || value.startsWith('git+ssh://')) {
    let hostname: string | undefined;
    try {
      hostname = new URL(value).hostname;
    } catch {
      // Keep malformed SSH URLs on the generic git path so clone reports the error.
    }

    return {
      type: hostname && isGitLabHost(hostname) ? 'gitlab' : 'git',
      url: value,
    };
  }

  return null;
}

export function parseSourceInput(value: string): ParsedSource | null {
  const normalizedValue = value.trim();
  const baseDir = process.env.INIT_CWD?.trim() || process.cwd();
  const localPath = resolve(baseDir, normalizedValue);

  if (existsSync(localPath)) {
    return {
      type: 'local',
      url: localPath,
      localPath,
    };
  }

  if (normalizedValue.startsWith('gitlab:')) {
    return parseGitLabPrefixSource(normalizedValue);
  }

  const httpSource = parseGitHttpSource(normalizedValue);
  if (httpSource) {
    return httpSource;
  }

  const sshSource = parseGitSshSource(normalizedValue);
  if (sshSource) {
    return sshSource;
  }

  // `git://` is not SSH; keep it on the generic git path.
  if (normalizedValue.startsWith('git://')) {
    return {
      type: 'git',
      url: normalizedValue,
    };
  }

  if (/^gitlab\./i.test(normalizedValue)) {
    return parseGitLabHostShorthand(normalizedValue);
  }

  const shorthandMatch = normalizedValue.match(GITHUB_SHORTHAND_RE);
  if (shorthandMatch?.groups) {
    const owner = shorthandMatch.groups.owner;
    const repo = shorthandMatch.groups.repo;
    const skillFilter = shorthandMatch.groups.skill?.trim();
    return {
      type: 'github',
      url: `https://github.com/${owner}/${repo}.git`,
      ...(skillFilter ? { skillFilter } : {}),
    };
  }

  return null;
}
