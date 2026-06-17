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
