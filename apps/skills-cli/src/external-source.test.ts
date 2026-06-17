import { describe, expect, it, vi, afterEach } from 'vitest';
import type { ParsedSource } from './types.ts';

vi.mock('./git.ts', () => ({
  cloneRepo: vi.fn(),
}));

vi.mock('./skills.ts', () => ({
  discoverSkills: vi.fn(),
  filterSkills: vi.fn((availableSkills, requestedNames) =>
    availableSkills.filter((skill: { name: string }) => requestedNames.includes(skill.name))
  ),
}));

import { cloneRepo } from './git.ts';
import { discoverSkills } from './skills.ts';
import { resolveExternalSourceSkills } from './external-source.ts';

describe('resolveExternalSourceSkills', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('uses a local source path without cloning', async () => {
    const source: ParsedSource = {
      type: 'local',
      url: '/tmp/local-skill',
      localPath: '/tmp/local-skill',
    };

    vi.mocked(discoverSkills).mockResolvedValueOnce([
      { name: 'zoom-out', description: 'Broader context', path: '/tmp/local-skill' },
    ]);

    const result = await resolveExternalSourceSkills(source, {});

    expect(cloneRepo).not.toHaveBeenCalled();
    expect(discoverSkills).toHaveBeenCalledWith('/tmp/local-skill', undefined, { fullDepth: false });
    expect(result.sourceRoot).toBe('/tmp/local-skill');
    expect(result.cleanupRoot).toBeNull();
    expect(result.matchedSkills.map((skill) => skill.name)).toEqual(['zoom-out']);
  });

  it('clones a remote source and merges explicit --skill names with source shorthand filter', async () => {
    const source: ParsedSource = {
      type: 'github',
      url: 'https://github.com/acme/skills.git',
      ref: 'main',
      skillFilter: 'zoom-out',
    };

    vi.mocked(cloneRepo).mockResolvedValueOnce('/tmp/cloned-skills');
    vi.mocked(discoverSkills).mockResolvedValueOnce([
      { name: 'zoom-out', description: 'Broader context', path: '/tmp/cloned-skills/zoom-out' },
      { name: 'pdf', description: 'PDF helper', path: '/tmp/cloned-skills/pdf' },
    ]);

    const result = await resolveExternalSourceSkills(source, {
      requestedSkillNames: ['pdf'],
      fullDepth: true,
    });

    expect(cloneRepo).toHaveBeenCalledWith('https://github.com/acme/skills.git', 'main');
    expect(discoverSkills).toHaveBeenCalledWith('/tmp/cloned-skills', undefined, { fullDepth: true });
    expect(result.requestedSkillNames).toEqual(['pdf', 'zoom-out']);
    expect(result.matchedSkills.map((skill) => skill.name)).toEqual(['zoom-out', 'pdf']);
    expect(result.cleanupRoot).toBe('/tmp/cloned-skills');
  });

  it('throws when the requested skills do not match any discovered skill', async () => {
    const source: ParsedSource = {
      type: 'github',
      url: 'https://github.com/acme/skills.git',
    };

    vi.mocked(cloneRepo).mockResolvedValueOnce('/tmp/cloned-skills');
    vi.mocked(discoverSkills).mockResolvedValueOnce([
      { name: 'zoom-out', description: 'Broader context', path: '/tmp/cloned-skills/zoom-out' },
    ]);

    await expect(
      resolveExternalSourceSkills(source, {
        requestedSkillNames: ['pdf'],
      })
    ).rejects.toThrow('No matching skills found for: pdf');
  });
});
