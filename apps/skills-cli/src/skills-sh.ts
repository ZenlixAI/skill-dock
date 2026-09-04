import { searchNpmRegistrySkills } from './npm-registry.ts';
import { sanitizeMetadata } from './sanitize.ts';
import type { SearchSkill } from './find.ts';

export async function searchSkillsShSkills(
  query: string,
  options: { registry?: string; size?: number } = {}
): Promise<SearchSkill[]> {
  const registryUrl = (options.registry?.trim() || 'https://skills.sh').replace(/\/+$/, '');
  const results = await searchNpmRegistrySkills(query, {
    registry: registryUrl,
    size: options.size ?? 10,
  });

  return results.map((item) => ({
    packageName: sanitizeMetadata(item.packageName),
    displayName: sanitizeMetadata(item.displayName),
    description: sanitizeMetadata(item.description || ''),
    latestVersion: item.latestVersion ? sanitizeMetadata(item.latestVersion) : undefined,
    sourceKind: 'skills.sh' as const,
    originRegistryUrl: registryUrl,
    learnMoreUrl: new URL(item.packageName, `${registryUrl}/`).toString(),
    installable: false,
    hosted: false,
    mirrored: false,
  }));
}
