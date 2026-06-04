import { createHash } from 'crypto';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { computeManifestDigest, inspectSkillDirectory } from './authoring.ts';

describe('authoring helpers', () => {
  let testDir: string;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-authoring-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });
  });

  afterEach(() => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('computes manifest digest from SKILL.md content only', async () => {
    const skillContent = `---
name: Probe Skill
description: Probe publish payload
---
`;
    writeFileSync(join(testDir, 'SKILL.md'), skillContent);
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/probe-skill', version: '0.1.0' }, null, 2)
    );
    writeFileSync(join(testDir, '.npmrc'), '//127.0.0.1:18080/:_authToken=fake-token');
    mkdirSync(join(testDir, 'commands'), { recursive: true });
    writeFileSync(join(testDir, 'commands', 'run.txt'), 'echo probe');

    const digest = await computeManifestDigest(testDir);

    const expected = `sha256:${createHash('sha256').update(skillContent).digest('hex')}`;
    expect(digest).toBe(expected);
  });

  it('surfaces display metadata and rejects package names outside the .skill scope pattern', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: Probe Skill
description: Probe publish payload
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@other.scope/probe-skill', version: '0.1.0' }, null, 2)
    );

    const inspection = await inspectSkillDirectory(testDir);

    expect(inspection.displayName).toBe('Probe Skill');
    expect(inspection.description).toBe('Probe publish payload');
    expect(inspection.ok).toBe(false);
    expect(inspection.errors).toContain(
      'package.json name must use a scoped @<company>.skill/<skill-name> package name'
    );
  });

  it('accepts company-specific .skill package scopes', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: Probe Skill
description: Probe publish payload
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@zhenlikeji.skill/probe-skill', version: '0.1.0' }, null, 2)
    );

    const inspection = await inspectSkillDirectory(testDir);

    expect(inspection.ok).toBe(true);
    expect(inspection.packageName).toBe('@zhenlikeji.skill/probe-skill');
  });

  it('reads package skill sourceType metadata when present', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: Probe Skill
description: Probe publish payload
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify(
        {
          name: '@platform.skill/probe-skill',
          version: '0.1.0',
          skill: {
            sourceType: 'platform_created',
          },
        },
        null,
        2
      )
    );

    const inspection = await inspectSkillDirectory(testDir);

    expect(inspection.ok).toBe(true);
    expect(inspection.sourceType).toBe('platform_created');
  });

  it('does not report empty capability directories as components', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: Probe Skill
description: Probe publish payload
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/probe-skill', version: '0.1.0' }, null, 2)
    );
    mkdirSync(join(testDir, 'docs'), { recursive: true });

    const inspection = await inspectSkillDirectory(testDir);

    expect(inspection.ok).toBe(true);
    expect(inspection.components).toEqual([]);
  });
});
