import { join, relative } from 'path';
import { parseSourceInput } from './source-parser.ts';
import type { ParsedSource, Skill } from './types.ts';

interface SyncLegacyShape {
  sourceType?: string;
  legacy?: {
    source?: string;
    ref?: string;
    skillPath?: string;
  };
}

export function getRecordedSkillPath(input: SyncLegacyShape): string | null {
  const raw = input.legacy?.skillPath?.trim();
  return raw ? raw.replace(/\\/g, '/') : null;
}

export function parseLegacyReplaySource(input: SyncLegacyShape): ParsedSource | null {
  const rawSource = input.legacy?.source?.trim();
  if (!rawSource) {
    return null;
  }

  const parsed = parseSourceInput(rawSource);
  if (!parsed) {
    return null;
  }

  return {
    ...parsed,
    ref: input.legacy?.ref?.trim() || parsed.ref,
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
