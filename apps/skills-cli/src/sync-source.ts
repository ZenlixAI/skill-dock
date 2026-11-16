import { join, relative } from 'path';
import { parseSourceInput } from './source-parser.ts';
import type { ParsedSource, Skill } from './types.ts';

interface SyncLegacyShape {
  sourceType?: string;
  legacy?: {
    source?: string;
    sourceUrl?: string;
    ref?: string;
    skillPath?: string;
  };
}

export function getRecordedSkillPath(input: SyncLegacyShape): string | null {
  const raw = input.legacy?.skillPath?.trim();
  return raw ? raw.replace(/\\/g, '/') : null;
}

export function parseLegacyReplaySource(input: SyncLegacyShape): ParsedSource | null {
  const recordedRef = input.legacy?.ref?.trim() || undefined;
  const sourceUrl = input.legacy?.sourceUrl?.trim();
  const source = input.legacy?.source?.trim();
  const parsedSourceUrl = sourceUrl ? parseSourceInput(sourceUrl) : null;
  const parsedSource = source ? parseSourceInput(source) : null;
  const base = parsedSourceUrl || parsedSource;

  if (!base) {
    return null;
  }

  // The normalized sourceUrl is the best clone target, but the original source
  // may still carry selectors such as ?path= or a GitLab /-/tree/ subpath.
  const subpath = base.subpath || parsedSource?.subpath;
  const ref = recordedRef || base.ref || parsedSource?.ref;

  return {
    ...base,
    ...(subpath ? { subpath } : {}),
    ...(ref ? { ref } : {}),
  };
}

export function selectDiscoveredSkillForSync(
  discoveredSkills: Skill[],
  searchRoot: string,
  recordedSkillPath: string
): Skill {
  const normalizedTarget = recordedSkillPath.replace(/\\/g, '/');
  const matches = discoveredSkills.filter((skill) => {
    const skillMdPath = join(skill.path, 'SKILL.md');
    return relative(searchRoot, skillMdPath).replace(/\\/g, '/') === normalizedTarget;
  });

  if (matches.length === 1) {
    return matches[0]!;
  }
  if (matches.length === 0) {
    throw new Error('Recorded skill path no longer resolves to a discovered skill.');
  }
  throw new Error('Recorded skill path resolved to multiple discovered skills.');
}
