import { existsSync } from 'fs';
import { resolve } from 'path';
import type { ParsedSource } from './types.ts';

const GITHUB_SHORTHAND_RE =
  /^(?<owner>[A-Za-z0-9_.-]+)\/(?<repo>[A-Za-z0-9_.-]+?)(?:@(?<skill>[^/?#]+))?$/;

function normalizeSourceSubpath(raw: string | null): string | undefined {
  if (!raw) return undefined;
  const normalized = raw.trim().replace(/^\/+/, '');
  return normalized.length > 0 ? normalized : undefined;
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

  if (parsed.pathname.endsWith('.git')) {
    return {
      type: 'git',
      url: value,
      ...(subpath ? { subpath } : {}),
      ...(ref ? { ref } : {}),
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

  const httpSource = parseGitHttpSource(normalizedValue);
  if (httpSource) {
    return httpSource;
  }

  if (
    normalizedValue.startsWith('git@') ||
    normalizedValue.startsWith('ssh://') ||
    normalizedValue.startsWith('git://') ||
    normalizedValue.startsWith('git+ssh://')
  ) {
    return {
      type: 'git',
      url: normalizedValue,
    };
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
