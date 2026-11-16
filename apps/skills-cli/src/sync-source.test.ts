import { describe, expect, it } from 'vitest';
import type { Skill } from './types.ts';
import { parseSourceInput } from './source-parser.ts';
import {
  getRecordedSkillPath,
  parseLegacyReplaySource,
  selectDiscoveredSkillForSync,
} from './sync-source.ts';

describe('sync source helpers', () => {
  it('replays a github legacy source with its ref', () => {
    const parsed = parseLegacyReplaySource({
      sourceType: 'github',
      legacy: {
        source: 'https://github.com/acme/skills.git',
        ref: 'main',
      },
    });

    expect(parsed).toMatchObject(parseSourceInput('https://github.com/acme/skills.git')!);
    expect(parsed?.ref).toBe('main');
  });

  it('prefers the normalized sourceUrl over the original source', () => {
    const parsed = parseLegacyReplaySource({
      sourceType: 'gitlab',
      legacy: {
        source: 'https://gitlab.com/acme/skills',
        sourceUrl: 'git@gitlab.com:acme/skills.git',
        ref: 'main',
      },
    });

    expect(parsed).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:acme/skills.git',
      ref: 'main',
    });
  });

  it('keeps selectors from the original source when replaying a normalized URL', () => {
    const parsed = parseLegacyReplaySource({
      sourceType: 'gitlab',
      legacy: {
        source: 'https://gitlab.com/acme/skills/-/tree/main/skills/pdf',
        sourceUrl: 'git@gitlab.com:acme/skills.git',
        ref: 'main',
      },
    });

    expect(parsed).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:acme/skills.git',
      ref: 'main',
      subpath: 'skills/pdf',
    });
  });

  it('falls back to source when sourceUrl cannot be parsed', () => {
    const parsed = parseLegacyReplaySource({
      sourceType: 'gitlab',
      legacy: {
        sourceUrl: 'https://example.com/not-a-repo',
        source: 'https://gitlab.com/acme/skills',
      },
    });

    expect(parsed).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:acme/skills.git',
    });
  });

  it('returns the recorded SKILL.md path from legacy metadata', () => {
    expect(
      getRecordedSkillPath({
        legacy: { skillPath: 'skills/legacy-skill/SKILL.md' },
      })
    ).toBe('skills/legacy-skill/SKILL.md');
  });

  it('selects the discovered skill whose SKILL.md matches legacy.skillPath', () => {
    const discovered: Skill[] = [
      { name: 'first', description: 'First', path: '/repo/skills/first' },
      { name: 'second', description: 'Second', path: '/repo/skills/second' },
    ];

    expect(
      selectDiscoveredSkillForSync(discovered, '/repo', 'skills/second/SKILL.md')
    ).toMatchObject({ name: 'second', path: '/repo/skills/second' });
  });

  it('throws when legacy.skillPath does not match any discovered skill', () => {
    expect(() =>
      selectDiscoveredSkillForSync(
        [{ name: 'first', description: 'First', path: '/repo/skills/first' }],
        '/repo',
        'skills/missing/SKILL.md'
      )
    ).toThrow('Recorded skill path no longer resolves to a discovered skill');
  });
});
