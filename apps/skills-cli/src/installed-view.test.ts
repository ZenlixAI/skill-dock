import { afterEach, describe, expect, it, vi } from 'vitest';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import * as agentsModule from './agents.ts';
import { listInstalledView } from './installed-view.ts';

describe('installed view projection', () => {
  const roots: string[] = [];

  afterEach(() => {
    vi.restoreAllMocks();
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

  it('marks a canonical disk install orphaned when no lock entry exists', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-missing-lock-'));
    roots.push(root);

    mkdirSync(join(root, '.agents', 'skills', 'manual-skill'), { recursive: true });
    writeFileSync(
      join(root, '.agents', 'skills', 'manual-skill', 'SKILL.md'),
      `---\nname: Manual Skill\ndescription: Installed outside the lock\n---\n`
    );

    const records = await listInstalledView({ scope: 'project', cwd: root });

    expect(records).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          displayName: 'Manual Skill',
          installName: 'manual-skill',
          orphaned: true,
          orphanReasons: expect.arrayContaining(['missing_lock_entry']),
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

    expect(records).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          displayName: 'pdf-skill',
          orphaned: true,
          orphanReasons: expect.arrayContaining(['alias_conflict']),
        }),
        expect.objectContaining({
          displayName: 'pdf skill',
          orphaned: true,
          orphanReasons: expect.arrayContaining(['alias_conflict']),
        }),
      ])
    );
  });

  it('does not mark healthy installs orphaned when only display aliases overlap', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-display-alias-'));
    roots.push(root);

    for (const dir of ['review-command', 'review-helper'] as const) {
      mkdirSync(join(root, '.agents', 'skills', dir), { recursive: true });
      writeFileSync(
        join(root, '.agents', 'skills', dir, 'SKILL.md'),
        `---\nname: Review\ndescription: Shared display alias\n---\n`
      );
    }

    writeFileSync(
      join(root, 'skills-lock.json'),
      JSON.stringify(
        {
          version: 1,
          skills: {
            '@platform.skill/review-command': {
              sourceType: 'registry',
              computedHash: 'hash-1',
              packageName: '@platform.skill/review-command',
              installName: 'review-command',
              version: '1.0.0',
            },
            '@platform.skill/review-helper': {
              sourceType: 'registry',
              computedHash: 'hash-2',
              packageName: '@platform.skill/review-helper',
              installName: 'review-helper',
              version: '1.0.0',
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
          packageName: '@platform.skill/review-command',
          installName: 'review-command',
          orphaned: false,
        }),
        expect.objectContaining({
          packageName: '@platform.skill/review-helper',
          installName: 'review-helper',
          orphaned: false,
        }),
      ])
    );
  });

  it('does not mark healthy installs orphaned when scoped packages share a basename alias', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-scoped-basename-'));
    roots.push(root);

    for (const dir of ['review-a', 'review-b'] as const) {
      mkdirSync(join(root, '.agents', 'skills', dir), { recursive: true });
      writeFileSync(
        join(root, '.agents', 'skills', dir, 'SKILL.md'),
        `---\nname: ${dir}\ndescription: Scoped package collision repro\n---\n`
      );
    }

    writeFileSync(
      join(root, 'skills-lock.json'),
      JSON.stringify(
        {
          version: 1,
          skills: {
            '@scope-a/review': {
              sourceType: 'registry',
              computedHash: 'hash-a',
              packageName: '@scope-a/review',
              installName: 'review-a',
              version: '1.0.0',
            },
            '@scope-b/review': {
              sourceType: 'registry',
              computedHash: 'hash-b',
              packageName: '@scope-b/review',
              installName: 'review-b',
              version: '1.0.0',
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
          packageName: '@scope-a/review',
          installName: 'review-a',
          orphaned: false,
        }),
        expect.objectContaining({
          packageName: '@scope-b/review',
          installName: 'review-b',
          orphaned: false,
        }),
      ])
    );
  });

  it('ignores stray agent directories without a valid SKILL.md', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-stray-agent-dir-'));
    roots.push(root);

    mkdirSync(join(root, '.claude', 'skills', 'stray-dir'), { recursive: true });
    writeFileSync(join(root, '.claude', 'skills', 'stray-dir', 'notes.txt'), 'not a skill');

    const records = await listInstalledView({ scope: 'project', cwd: root });

    expect(records).toEqual([]);
  });

  it('does not attach an unrelated agent-specific skill to a canonical install', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-unrelated-agent-skill-'));
    roots.push(root);

    mkdirSync(join(root, '.agents', 'skills', 'alpha-skill'), { recursive: true });
    writeFileSync(
      join(root, '.agents', 'skills', 'alpha-skill', 'SKILL.md'),
      `---\nname: Alpha\ndescription: Canonical alpha\n---\n`
    );
    writeFileSync(
      join(root, 'skills-lock.json'),
      JSON.stringify(
        {
          version: 1,
          skills: {
            '@scope/alpha': {
              sourceType: 'registry',
              computedHash: 'alpha-hash',
              packageName: '@scope/alpha',
              installName: 'alpha-skill',
              version: '1.0.0',
            },
          },
        },
        null,
        2
      )
    );

    mkdirSync(join(root, '.claude', 'skills', 'alpha'), { recursive: true });
    writeFileSync(
      join(root, '.claude', 'skills', 'alpha', 'SKILL.md'),
      `---\nname: Beta\ndescription: Unrelated beta skill\n---\n`
    );

    const records = await listInstalledView({ scope: 'project', cwd: root });

    expect(records).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          packageName: '@scope/alpha',
          agentLinks: expect.not.arrayContaining([
            expect.objectContaining({
              agent: 'claude-code',
              path: join(root, '.claude', 'skills', 'alpha'),
            }),
          ]),
        }),
        expect.objectContaining({
          displayName: 'Beta',
          installName: 'alpha',
          orphaned: true,
          orphanReasons: expect.arrayContaining(['missing_canonical_path']),
          agentLinks: expect.arrayContaining([
            expect.objectContaining({
              agent: 'claude-code',
              path: join(root, '.claude', 'skills', 'alpha'),
            }),
          ]),
        }),
      ])
    );
  });

  it('honors agentFilter when projecting agent-specific links', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-agent-filter-'));
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
            },
          },
        },
        null,
        2
      )
    );
    mkdirSync(join(root, '.claude', 'skills', 'pdf-skill'), { recursive: true });

    const records = await listInstalledView({
      scope: 'project',
      cwd: root,
      agentFilter: ['cursor'],
    });

    expect(records).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          packageName: '@platform.skill/pdf-skill',
          agentLinks: expect.not.arrayContaining([
            expect.objectContaining({
              agent: 'claude-code',
            }),
          ]),
        }),
      ])
    );
  });

  it('does not invent universal agent links for filtered agents that are not detected', async () => {
    const root = mkdtempSync(join(tmpdir(), 'installed-view-filtered-universal-'));
    roots.push(root);

    vi.spyOn(agentsModule, 'detectInstalledAgents').mockResolvedValue([]);

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
            },
          },
        },
        null,
        2
      )
    );

    const records = await listInstalledView({
      scope: 'project',
      cwd: root,
      agentFilter: ['cursor'],
    });

    expect(records).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          packageName: '@platform.skill/pdf-skill',
          agentLinks: expect.not.arrayContaining([
            expect.objectContaining({
              agent: 'cursor',
            }),
          ]),
        }),
      ])
    );
  });
});
