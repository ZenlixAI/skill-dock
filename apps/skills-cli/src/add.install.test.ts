import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as prompts from '@clack/prompts';
import { runAdd } from './add.ts';
import * as workflow from './registry-install-workflow.ts';
import * as agentRegistry from './agents.ts';

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
  materializeRegistrySkill: vi.fn(() =>
    Promise.resolve({
      packageName: '@platform.skill/pdf-skill',
      resolvedVersion: '1.2.0',
      installName: 'pdf-skill',
      integrity: 'sha512-test',
    })
  ),
  runRegistryInstallWorkflow: vi.fn(() =>
    Promise.resolve({
      packageName: '@platform.skill/pdf-skill',
      target: { version: '1.2.0' },
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
    })
  ),
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
    cursor: {
      displayName: 'Cursor',
      skillsDir: '.agents/skills',
      globalSkillsDir: '/tmp/.cursor/skills',
    },
  },
  detectInstalledAgents: vi.fn(() => Promise.resolve(['claude-code'])),
  getUniversalAgents: vi.fn(() => ['codex', 'cursor']),
  getNonUniversalAgents: vi.fn(() => []),
  isUniversalAgent: vi.fn((agent: string) => agent !== 'claude-code'),
}));

vi.mock('./installer.ts', () => ({
  getCanonicalPath: vi.fn(() => '/tmp/.agents/skills/pdf-skill'),
}));

vi.mock('./detect-agent.ts', () => ({
  detectAgent: vi.fn(() => Promise.resolve({ isAgent: false })),
  getAgentType: vi.fn(),
}));

vi.mock('./skill-lock.ts', () => ({
  getLastSelectedAgents: vi.fn(() => Promise.resolve(undefined)),
  saveSelectedAgents: vi.fn(),
}));

vi.mock('./telemetry.ts', () => ({
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
  },
}));

describe('runAdd registry installs', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('delegates registry installs to the shared workflow', async () => {
    await runAdd(['@platform.skill/pdf-skill'], {
      global: true,
      yes: true,
      agent: ['claude-code'],
    });

    expect(workflow.runRegistryInstallWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({
        packageName: '@platform.skill/pdf-skill',
        installGlobally: true,
        targetAgents: ['claude-code'],
        installMode: 'copy',
      })
    );
  });

  it('uses universal agents by default for non-interactive installs when no agents are detected', async () => {
    vi.mocked(agentRegistry.detectInstalledAgents).mockResolvedValueOnce([]);

    await runAdd(['@platform.skill/pdf-skill'], {
      yes: true,
    });

    expect(workflow.runRegistryInstallWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({
        targetAgents: ['codex', 'cursor'],
      })
    );
  });

  it('materializes an explicit output directory without selecting agents', async () => {
    const log = vi.spyOn(console, 'log').mockImplementation(() => {});
    try {
      await runAdd(['@platform.skill/pdf-skill'], {
        outputDir: '/tmp/sidecar-cache/item-000',
        json: true,
        version: '1.2.0',
      });

    expect(workflow.materializeRegistrySkill).toHaveBeenCalledWith({
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      registry: undefined,
      target: expect.objectContaining({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
      }),
      outputDir: '/tmp/sidecar-cache/item-000',
      cwd: process.cwd(),
    });
      expect(workflow.runRegistryInstallWorkflow).not.toHaveBeenCalled();
      expect(log).toHaveBeenCalledWith(expect.stringContaining('"installName":"pdf-skill"'));
    } finally {
      log.mockRestore();
    }
  });

  it('shows a compact success summary by default', async () => {
    vi.mocked(workflow.runRegistryInstallWorkflow).mockResolvedValueOnce({
      packageName: '@platform.skill/pdf-skill',
      target: { version: '1.2.0' } as never,
      successful: [
        {
          agent: 'claude-code',
          result: {
            success: true,
            path: '/tmp/.claude/skills/pdf-skill',
            canonicalPath: '/tmp/.agents/skills/pdf-skill',
            mode: 'symlink',
          },
        },
        {
          agent: 'codex',
          result: {
            success: true,
            path: '/tmp/.agents/skills/pdf-skill',
            canonicalPath: '/tmp/.agents/skills/pdf-skill',
            mode: 'symlink',
          },
        },
      ],
      failed: [],
    });

    await runAdd(['@platform.skill/pdf-skill'], {
      global: true,
      yes: true,
      agent: ['claude-code', 'codex'],
    });

    expect(vi.mocked(prompts.note).mock.calls[1]?.[1]).toContain('Installed 2 targets');
    expect(vi.mocked(prompts.note).mock.calls[1]?.[0]).toContain('Agents:');
    expect(vi.mocked(prompts.note).mock.calls[1]?.[0]).toContain('Location:');
    expect(vi.mocked(prompts.note).mock.calls[1]?.[0]).not.toContain('✓');
  });

  it('shows per-agent install details in verbose mode', async () => {
    vi.mocked(workflow.runRegistryInstallWorkflow).mockResolvedValueOnce({
      packageName: '@platform.skill/pdf-skill',
      target: { version: '1.2.0' } as never,
      successful: [
        {
          agent: 'claude-code',
          result: {
            success: true,
            path: '/tmp/.claude/skills/pdf-skill',
            canonicalPath: '/tmp/.agents/skills/pdf-skill',
            mode: 'symlink',
          },
        },
      ],
      failed: [],
    });

    await runAdd(['@platform.skill/pdf-skill'], {
      global: true,
      yes: true,
      verbose: true,
      agent: ['claude-code'],
    });

    expect(vi.mocked(prompts.note).mock.calls[1]?.[0]).toContain('✓');
    expect(vi.mocked(prompts.note).mock.calls[1]?.[0]).toContain('Claude Code');
  });
});
