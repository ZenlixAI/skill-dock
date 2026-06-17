import { cloneRepo } from './git.ts';
import { discoverSkills, filterSkills } from './skills.ts';
import type { ParsedSource, Skill } from './types.ts';

export interface ResolveExternalSourceSkillsOptions {
  requestedSkillNames?: string[];
  fullDepth?: boolean;
}

export interface ResolvedExternalSourceSkills {
  sourceRoot: string;
  cleanupRoot: string | null;
  discoveredSkills: Skill[];
  requestedSkillNames: string[];
  matchedSkills: Skill[];
}

export async function resolveExternalSourceSkills(
  source: ParsedSource,
  options: ResolveExternalSourceSkillsOptions
): Promise<ResolvedExternalSourceSkills> {
  const sourceRoot =
    source.type === 'local' ? source.localPath || source.url : await cloneRepo(source.url, source.ref);
  const cleanupRoot = source.type === 'local' ? null : sourceRoot;

  const discoveredSkills = await discoverSkills(sourceRoot, source.subpath, {
    fullDepth: options.fullDepth === true,
  });

  if (discoveredSkills.length === 0) {
    throw new Error('No skills found in the provided source.');
  }

  const requestedSkillNames = [
    ...(options.requestedSkillNames || []),
    ...(source.skillFilter ? [source.skillFilter] : []),
  ];

  const matchedSkills =
    requestedSkillNames.length > 0
      ? filterSkills(discoveredSkills, requestedSkillNames)
      : discoveredSkills;

  if (requestedSkillNames.length > 0 && matchedSkills.length === 0) {
    throw new Error(`No matching skills found for: ${requestedSkillNames.join(', ')}`);
  }

  return {
    sourceRoot,
    cleanupRoot,
    discoveredSkills,
    requestedSkillNames,
    matchedSkills,
  };
}
