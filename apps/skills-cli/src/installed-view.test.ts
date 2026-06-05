import { afterEach, describe, expect, it } from 'vitest';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { listInstalledView } from './installed-view.ts';

describe('installed view projection', () => {
  const roots: string[] = [];

  afterEach(() => {
    for (const root of roots.splice(0)) {
      rmSync(root, { recursive: true, force: true });
    }
  });

  it('projects a tracked project registry install with aliases and agent links', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-project-'));
    roots.push(root);

    mkdirSync(join(root, '.agents', 'skills', 'pdf-skill'), { recursive: true });
    writeFileSync(
      join(root, '.agents', 'skills', 'pdf-skill', 'SKILL.md'),
      `---\nname: PDF Skill\ndescription: Read PDFs\n---\n`
    );
    writeFileSync(
      join(root, 'skills-lock.json'),
      JSON.stringify(
        {
          version: 1,
          skills: {
            '@platform.skill/pdf-skill': {
              sourceType: 'registry',
              computedHash: 'abc123',
              packageName: '@platform.skill/pdf-skill',
              installName: 'pdf-skill',
              version: '1.2.3',
              registry: 'https://catalog.example.com',
            },
          },
        },
        null,
        2
      )
    );
    mkdirSync(join(root, '.claude', 'skills', 'pdf-skill'), { recursive: true });

    const records = await listInstalledView({ scope: 'project', cwd: root });

    expect(records).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          packageName: '@platform.skill/pdf-skill',
          installName: 'pdf-skill',
          aliases: expect.arrayContaining(['@platform.skill/pdf-skill', 'pdf-skill']),
          agentLinks: expect.arrayContaining([
            expect.objectContaining({
              agent: 'claude-code',
              path: join(root, '.claude', 'skills', 'pdf-skill'),
            }),
          ]),
          scope: 'project',
          orphaned: false,
        }),
      ])
    );
  });

  it('marks a skill orphaned when only the agent-specific directory exists', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-agent-only-'));
    roots.push(root);

    mkdirSync(join(root, '.claude', 'skills', 'pdf-skill'), { recursive: true });
    writeFileSync(
      join(root, '.claude', 'skills', 'pdf-skill', 'SKILL.md'),
      `---\nname: PDF Skill\ndescription: Read PDFs\n---\n`
    );

    const records = await listInstalledView({ scope: 'project', cwd: root });

    expect(records).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          displayName: 'PDF Skill',
          orphaned: true,
          orphanReasons: expect.arrayContaining(['missing_canonical_path']),
        }),
      ])
    );
  });

  it('marks a lock-only registry entry orphaned when the canonical path is missing', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-lock-only-'));
    roots.push(root);

    writeFileSync(
      join(root, 'skills-lock.json'),
      JSON.stringify(
        {
          version: 1,
          skills: {
            '@platform.skill/pdf-skill': {
              sourceType: 'registry',
              computedHash: 'abc123',
              packageName: '@platform.skill/pdf-skill',
              installName: 'pdf-skill',
              version: '1.2.3',
            },
          },
        },
        null,
        2
      )
    );

    const records = await listInstalledView({ scope: 'project', cwd: root });

    expect(records).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          packageName: '@platform.skill/pdf-skill',
          orphaned: true,
          orphanReasons: expect.arrayContaining(['missing_canonical_path']),
        }),
      ])
    );
  });

  it('marks conflicting install aliases as orphaned', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-conflict-'));
    roots.push(root);

    for (const dir of ['pdf-skill', 'pdf skill']) {
      mkdirSync(join(root, '.agents', 'skills', dir), { recursive: true });
      writeFileSync(
        join(root, '.agents', 'skills', dir, 'SKILL.md'),
        `---\nname: ${dir}\ndescription: collision\n---\n`
      );
    }

    const records = await listInstalledView({ scope: 'project', cwd: root });

    expect(records.some((record) => record.orphanReasons.includes('alias_conflict'))).toBe(true);
  });
});
