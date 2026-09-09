import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mkdirSync, rmSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runAdd } from './add.ts';
import * as workflow from './registry-install-workflow.ts';
import * as installer from './installer.ts';
import * as skills from './skills.ts';
import * as git from './git.ts';
import * as localLock from './local-lock.ts';

vi.mock('./catalog.ts', () => ({
  resolveInstallTarget: vi.fn(async (packageName: string) => ({
    packageName,
    version: '1.2.0',
    npmRegistry: 'https://npm.example.com',
    tarballUrl:
      'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
    integrity: 'sha512-test',
    skillManifestDigest: 'sha256-manifest',
    versionStatus: 'published',
    registryAccessMode: 'proxy',
  })),
}));

vi.mock('./registry-install-workflow.ts', () => ({
  runRegistryInstallWorkflow: vi.fn(),
}));

vi.mock('./skills.ts', () => ({
  discoverSkills: vi.fn(),
  filterSkills: vi.fn((availableSkills, requestedNames) =>
    availableSkills.filter((skill: { name: string }) => requestedNames.includes(skill.name))
  ),
}));

vi.mock('./git.ts', () => ({
  cloneRepo: vi.fn(),
  cleanupTempDir: vi.fn(() => Promise.resolve()),
}));

vi.mock('./add-sync.ts', () => ({
  syncAddedSkillToBackend: vi.fn(() => Promise.resolve(null)),
}));

vi.mock('./local-lock.ts', () => ({
  addSkillToLocalLock: vi.fn(() => Promise.resolve()),
  computeSkillFolderHash: vi.fn(() => Promise.resolve('computed-hash')),
}));

vi.mock('./skill-lock.ts', () => ({
  addSkillToLock: vi.fn(() => Promise.resolve()),
  getLastSelectedAgents: vi.fn(() => Promise.resolve(undefined)),
  saveSelectedAgents: vi.fn(),
}));

vi.mock('./agents.ts', () => ({
  agents: {
    'claude-code': {
      displayName: 'Claude Code',
      skillsDir: '.claude/skills',
      globalSkillsDir: '/tmp/.claude/skills',
    },
    codex: {
      displayName: 'Codex',
      skillsDir: '.agents/skills',
      globalSkillsDir: '/tmp/.codex/skills',
    },
  },
  detectInstalledAgents: vi.fn(() => Promise.resolve(['claude-code'])),
  getUniversalAgents: vi.fn(() => ['codex']),
  getNonUniversalAgents: vi.fn(() => []),
  isUniversalAgent: vi.fn((agent: string) => agent === 'codex'),
}));

vi.mock('./installer.ts', () => ({
  getCanonicalPath: vi.fn((name: string) => `/tmp/.agents/skills/${name}`),
  installSkillForAgent: vi.fn(() =>
    Promise.resolve({
      success: true,
      path: '/tmp/.agents/skills/zoom-out',
      canonicalPath: '/tmp/.agents/skills/zoom-out',
      mode: 'copy',
    })
  ),
}));

vi.mock('./detect-agent.ts', () => ({
  detectAgent: vi.fn(() => Promise.resolve({ isAgent: false })),
  getAgentType: vi.fn(),
}));

vi.mock('./telemetry.ts', () => ({
  track: vi.fn(),
  setVersion: vi.fn(),
}));

vi.mock('@clack/prompts', () => ({
  intro: vi.fn(),
  outro: vi.fn(),
  note: vi.fn(),
  confirm: vi.fn(() => Promise.resolve(true)),
  isCancel: vi.fn(() => false),
  spinner: vi.fn(() => ({
    start: vi.fn(),
    stop: vi.fn(),
  })),
  log: {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    message: vi.fn(),
    step: vi.fn(),
    success: vi.fn(),
  },
}));

describe('runAdd source installs', () => {
  let localSourceDir: string;
  let remoteRepoDir: string;

  beforeEach(() => {
    vi.clearAllMocks();
    localSourceDir = join(tmpdir(), `skills-add-source-${Date.now()}`);
    remoteRepoDir = join(tmpdir(), `skills-add-remote-${Date.now()}`);
    mkdirSync(join(localSourceDir, 'skills', 'zoom-out'), { recursive: true });
    mkdirSync(join(remoteRepoDir, 'skills', 'zoom-out'), { recursive: true });
    writeFileSync(
      join(localSourceDir, 'skills', 'zoom-out', 'SKILL.md'),
      ['---', 'name: zoom-out', 'description: Broader context', '---', '', '# zoom-out'].join('\n')
    );
    writeFileSync(
      join(localSourceDir, 'skills', 'zoom-out', 'package.json'),
      JSON.stringify({ name: '@acme.skill/zoom-out', version: '0.1.0' }, null, 2)
    );
    writeFileSync(
      join(remoteRepoDir, 'skills', 'zoom-out', 'SKILL.md'),
      ['---', 'name: zoom-out', 'description: Broader context', '---', '', '# zoom-out'].join('\n')
    );
    writeFileSync(
      join(remoteRepoDir, 'skills', 'zoom-out', 'package.json'),
      JSON.stringify({ name: '@acme.skill/zoom-out', version: '0.1.0' }, null, 2)
    );
    vi.mocked(skills.discoverSkills).mockResolvedValue([
      {
        name: 'zoom-out',
        description: 'Broader context',
        path: join(localSourceDir, 'skills', 'zoom-out'),
        packageName: '@acme.skill/zoom-out',
      },
    ]);
  });

  afterEach(() => {
    rmSync(localSourceDir, { recursive: true, force: true });
    rmSync(remoteRepoDir, { recursive: true, force: true });
  });

  it('installs a skill from a local path without using the registry workflow', async () => {
    await runAdd([localSourceDir], {
      yes: true,
      agent: ['claude-code'],
    });

    expect(workflow.runRegistryInstallWorkflow).not.toHaveBeenCalled();
    expect(skills.discoverSkills).toHaveBeenCalledWith(localSourceDir, undefined, {
      fullDepth: false,
    });
    expect(installer.installSkillForAgent).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'zoom-out' }),
      'claude-code',
      expect.objectContaining({ mode: 'copy' })
    );
    expect(localLock.addSkillToLocalLock).toHaveBeenCalledWith(
      'zoom-out',
      expect.objectContaining({
        sourceType: 'local',
        computedHash: 'computed-hash',
      }),
      expect.any(String)
    );
  });

  it('clones a GitHub shorthand source, installs discovered skills, and cleans up', async () => {
    vi.mocked(git.cloneRepo).mockResolvedValue('/tmp/cloned-owner-repo');

    await runAdd(['owner/repo'], {
      yes: true,
      agent: ['claude-code'],
    });

    expect(git.cloneRepo).toHaveBeenCalledWith('https://github.com/owner/repo.git', undefined);
    expect(skills.discoverSkills).toHaveBeenCalledWith('/tmp/cloned-owner-repo', undefined, {
      fullDepth: false,
    });
    expect(localLock.addSkillToLocalLock).toHaveBeenCalledWith(
      'zoom-out',
      expect.objectContaining({
        sourceType: 'github',
        computedHash: 'computed-hash',
      }),
      expect.any(String)
    );
    expect(git.cleanupTempDir).toHaveBeenCalledWith('/tmp/cloned-owner-repo');
  });

  it('still routes registry package names through the registry workflow first', async () => {
    vi.mocked(workflow.runRegistryInstallWorkflow).mockResolvedValue({
      packageName: '@platform.skill/pdf-skill',
      target: { version: '1.2.0' } as never,
      successful: [
        {
          agent: 'claude-code',
          result: {
            success: true,
            path: '/tmp/.agents/skills/pdf-skill',
            canonicalPath: '/tmp/.agents/skills/pdf-skill',
            mode: 'copy',
          },
        },
      ],
      failed: [],
    });

    await runAdd(['@platform.skill/pdf-skill'], {
      yes: true,
      agent: ['claude-code'],
    });

    expect(workflow.runRegistryInstallWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({
        packageName: '@platform.skill/pdf-skill',
      })
    );
    expect(skills.discoverSkills).not.toHaveBeenCalled();
    expect(installer.installSkillForAgent).not.toHaveBeenCalled();
  });
});
