import { existsSync, mkdirSync, rmSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./installed-view.ts', () => ({
  listInstalledView: vi.fn(),
}));
vi.mock('./git.ts', () => ({
  cloneRepo: vi.fn(),
  cleanupTempDir: vi.fn(() => Promise.resolve()),
}));
vi.mock('./skills.ts', () => ({
  discoverSkills: vi.fn(),
}));
vi.mock('./installer.ts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./installer.ts')>();
  return {
    ...actual,
    installSkillForAgent: vi.fn(),
  };
});
vi.mock('./local-lock.ts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./local-lock.ts')>();
  return {
    ...actual,
    addSkillToLocalLock: vi.fn(),
    computeSkillFolderHash: vi.fn(),
  };
});
vi.mock('./skill-lock.ts', () => ({
  addSkillToLock: vi.fn(),
}));

import { runSyncWorkflow } from './sync-workflow.ts';
import { cloneRepo, cleanupTempDir } from './git.ts';
import { listInstalledView } from './installed-view.ts';
import { installSkillForAgent } from './installer.ts';
import { addSkillToLocalLock, computeSkillFolderHash } from './local-lock.ts';
import { addSkillToLock } from './skill-lock.ts';
import { discoverSkills } from './skills.ts';

describe('sync workflow', () => {
  let localSourceDir: string;
  let clonedRoot: string;

  beforeEach(() => {
    localSourceDir = join(tmpdir(), `skills-sync-local-${Date.now()}`);
    clonedRoot = join(tmpdir(), `skills-sync-cloned-${Date.now()}`);
    mkdirSync(localSourceDir, { recursive: true });
    mkdirSync(join(clonedRoot, 'skills', 'legacy-skill'), { recursive: true });

    vi.mocked(listInstalledView).mockReset();
    vi.mocked(cloneRepo).mockReset();
    vi.mocked(cleanupTempDir).mockClear();
    vi.mocked(discoverSkills).mockReset();
    vi.mocked(installSkillForAgent).mockReset();
    vi.mocked(addSkillToLocalLock).mockReset();
    vi.mocked(computeSkillFolderHash).mockReset();
    vi.mocked(addSkillToLock).mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    if (existsSync(localSourceDir)) {
      rmSync(localSourceDir, { recursive: true, force: true });
    }
    if (existsSync(clonedRoot)) {
      rmSync(clonedRoot, { recursive: true, force: true });
    }
  });

  it('replays a project local-source install and refreshes the local lock entry', async () => {
    vi.mocked(listInstalledView).mockResolvedValueOnce([
      {
        lockKey: 'legacy-skill',
        packageName: undefined,
        displayName: 'Legacy Skill',
        installName: 'legacy-skill',
        aliases: ['legacy-skill'],
        scope: 'project',
        canonicalPath: '/repo/.agents/skills/legacy-skill',
        agentLinks: [{ agent: 'codex', path: '/repo/.agents/skills/legacy-skill', kind: 'canonical' }],
        sourceType: 'local',
        registry: undefined,
        version: undefined,
        pluginName: undefined,
        orphaned: false,
        orphanReasons: [],
        lockEntry: {
          sourceType: 'local',
          computedHash: 'old-hash',
          legacy: {
            source: localSourceDir,
            skillPath: 'SKILL.md',
          },
        },
      },
    ]);
    vi.mocked(discoverSkills).mockResolvedValueOnce([
      {
        name: 'Legacy Skill',
        description: 'Updated',
        path: localSourceDir,
      },
    ]);
    vi.mocked(installSkillForAgent).mockResolvedValueOnce({
      success: true,
      path: '/repo/.agents/skills/legacy-skill',
      canonicalPath: '/repo/.agents/skills/legacy-skill',
      mode: 'symlink',
    });
    vi.mocked(computeSkillFolderHash).mockResolvedValueOnce('new-hash');

    const result = await runSyncWorkflow({ project: true, skills: ['legacy-skill'] });

    expect(discoverSkills).toHaveBeenCalledWith(localSourceDir, undefined, { fullDepth: false });
    expect(installSkillForAgent).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Legacy Skill', path: localSourceDir }),
      'codex',
      expect.objectContaining({ global: false })
    );
    expect(addSkillToLocalLock).toHaveBeenCalledWith(
      'legacy-skill',
      expect.objectContaining({
        sourceType: 'local',
        computedHash: 'new-hash',
      }),
      expect.any(String)
    );
    expect(result).toMatchObject({
      successCount: 1,
      failCount: 0,
      skippedCount: 0,
    });
  });

  it('skips orphaned or registry-installed records', async () => {
    vi.mocked(listInstalledView).mockResolvedValueOnce([
      {
        lockKey: '@acme.skill/pdf',
        packageName: '@acme.skill/pdf',
        displayName: '@acme.skill/pdf',
        installName: 'pdf',
        aliases: ['pdf', '@acme.skill/pdf'],
        scope: 'project',
        canonicalPath: '/repo/.agents/skills/pdf',
        agentLinks: [],
        sourceType: 'registry',
        registry: 'https://catalog.example.com',
        version: '1.0.0',
        pluginName: undefined,
        orphaned: false,
        orphanReasons: [],
        lockEntry: {
          sourceType: 'registry',
          computedHash: 'hash',
          packageName: '@acme.skill/pdf',
          version: '1.0.0',
        },
      },
      {
        lockKey: 'broken-skill',
        packageName: undefined,
        displayName: 'broken-skill',
        installName: 'broken-skill',
        aliases: ['broken-skill'],
        scope: 'project',
        canonicalPath: '/repo/.agents/skills/broken-skill',
        agentLinks: [],
        sourceType: 'git',
        registry: undefined,
        version: undefined,
        pluginName: undefined,
        orphaned: true,
        orphanReasons: ['missing_canonical_path'],
        lockEntry: {
          sourceType: 'git',
          computedHash: 'hash',
          legacy: {
            source: 'https://github.com/acme/skills.git',
            skillPath: 'skills/broken-skill/SKILL.md',
          },
        },
      },
    ]);

    const result = await runSyncWorkflow({ project: true });

    expect(result).toMatchObject({
      successCount: 0,
      failCount: 0,
      skippedCount: 2,
    });
  });

  it('fails when replay metadata is incomplete', async () => {
    vi.mocked(listInstalledView).mockResolvedValueOnce([
      {
        lockKey: 'legacy-skill',
        packageName: undefined,
        displayName: 'Legacy Skill',
        installName: 'legacy-skill',
        aliases: ['legacy-skill'],
        scope: 'project',
        canonicalPath: '/repo/.agents/skills/legacy-skill',
        agentLinks: [{ agent: 'codex', path: '/repo/.agents/skills/legacy-skill', kind: 'canonical' }],
        sourceType: 'git',
        registry: undefined,
        version: undefined,
        pluginName: undefined,
        orphaned: false,
        orphanReasons: [],
        lockEntry: {
          sourceType: 'git',
          computedHash: 'old-hash',
          legacy: {
            skillPath: 'skills/legacy-skill/SKILL.md',
          },
        },
      },
    ]);

    const result = await runSyncWorkflow({ project: true, skills: ['legacy-skill'] });

    expect(result.successCount).toBe(0);
    expect(result.failCount).toBe(1);
    expect(result.failures[0]).toMatchObject({
      skill: 'Legacy Skill',
      error: 'legacy.source and legacy.skillPath are required for sync',
    });
  });

  it('clones git sources, refreshes the global lock hash, and always cleans up temp directories', async () => {
    vi.mocked(listInstalledView).mockResolvedValueOnce([
      {
        lockKey: 'legacy-skill',
        packageName: undefined,
        displayName: 'Legacy Skill',
        installName: 'legacy-skill',
        aliases: ['legacy-skill'],
        scope: 'global',
        canonicalPath: '/Users/test/.agents/skills/legacy-skill',
        agentLinks: [{ agent: 'codex', path: '/Users/test/.agents/skills/legacy-skill', kind: 'canonical' }],
        sourceType: 'git',
        registry: undefined,
        version: undefined,
        pluginName: undefined,
        orphaned: false,
        orphanReasons: [],
        lockEntry: {
          sourceType: 'git',
          skillFolderHash: 'old-hash',
          installedAt: '2026-06-16T00:00:00.000Z',
          updatedAt: '2026-06-16T00:00:00.000Z',
          legacy: {
            source: 'https://github.com/acme/skills.git',
            skillPath: 'skills/legacy-skill/SKILL.md',
          },
        },
      },
    ]);
    vi.mocked(cloneRepo).mockResolvedValueOnce(clonedRoot);
    vi.mocked(discoverSkills).mockResolvedValueOnce([
      {
        name: 'Legacy Skill',
        description: 'Updated',
        path: join(clonedRoot, 'skills', 'legacy-skill'),
      },
    ]);
    vi.mocked(installSkillForAgent).mockResolvedValueOnce({
      success: true,
      path: '/Users/test/.agents/skills/legacy-skill',
      canonicalPath: '/Users/test/.agents/skills/legacy-skill',
      mode: 'symlink',
    });
    vi.mocked(computeSkillFolderHash).mockResolvedValueOnce('global-hash');

    const result = await runSyncWorkflow({ global: true, yes: true });

    expect(cloneRepo).toHaveBeenCalledWith('https://github.com/acme/skills.git', undefined);
    expect(addSkillToLock).toHaveBeenCalledWith(
      'legacy-skill',
      expect.objectContaining({
        sourceType: 'git',
        skillFolderHash: 'global-hash',
      })
    );
    expect(cleanupTempDir).toHaveBeenCalledWith(clonedRoot);
    expect(result).toMatchObject({
      successCount: 1,
      failCount: 0,
      skippedCount: 0,
    });
  });
});
