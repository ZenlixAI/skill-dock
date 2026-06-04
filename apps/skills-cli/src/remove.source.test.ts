import { beforeEach, describe, expect, it, vi } from 'vitest';
import { removeCommand } from './remove.ts';
import * as telemetry from './telemetry.ts';
import * as localLock from './local-lock.ts';

vi.mock('./detect-agent.ts', () => ({
  detectAgent: vi.fn(() => Promise.resolve({ isAgent: false })),
}));

vi.mock('./agents.ts', () => ({
  agents: {
    'claude-code': {
      displayName: 'Claude Code',
      skillsDir: '.claude/skills',
    },
  },
  detectInstalledAgents: vi.fn(() => Promise.resolve([])),
}));

vi.mock('./telemetry.ts', () => ({
  track: vi.fn(),
}));

vi.mock('./skill-lock.ts', () => ({
  getAllLockedSkills: vi.fn(() => Promise.resolve({})),
  removeSkillFromLock: vi.fn(),
  getSkillFromLock: vi.fn(() => Promise.resolve(null)),
}));

vi.mock('./local-lock.ts', () => ({
  readLocalLock: vi.fn(),
  removeSkillFromLocalLock: vi.fn(() => Promise.resolve(true)),
}));

vi.mock('./installer.ts', () => ({
  getInstallPath: vi.fn(() => '/tmp/project/.claude/skills/platform.skill-pdf-skill'),
  getCanonicalPath: vi.fn(() => '/tmp/project/.agents/skills/platform.skill-pdf-skill'),
  getCanonicalSkillsDir: vi.fn(() => '/tmp/project/.agents/skills'),
  deriveInstallName: vi.fn((value: string) => value.split('/').pop() || value),
  getInstalledDirectoryNames: vi.fn(() => ['platform.skill-pdf-skill']),
  sanitizeName: vi.fn((value: string) => value.replace(/^@/, '').replace(/\//g, '-')),
}));

vi.mock('fs/promises', () => ({
  readdir: vi.fn((dir: string) => {
    if (dir.endsWith('/.agents/skills')) {
      return Promise.resolve([
        {
          name: 'platform.skill-pdf-skill',
          isDirectory: () => true,
        },
      ]);
    }
    return Promise.resolve([]);
  }),
  rm: vi.fn(() => Promise.resolve()),
  lstat: vi.fn(() => Promise.resolve({})),
}));

vi.mock('@clack/prompts', () => ({
  spinner: vi.fn(() => ({
    start: vi.fn(),
    stop: vi.fn(),
  })),
  outro: vi.fn(),
  log: {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    message: vi.fn(),
    success: vi.fn(),
  },
  confirm: vi.fn(() => Promise.resolve(true)),
  multiselect: vi.fn(),
  isCancel: vi.fn(() => false),
  cancel: vi.fn(),
}));

describe('removeCommand telemetry', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(localLock.readLocalLock).mockResolvedValue({
      version: 1,
      skills: {
        '@platform.skill/pdf-skill': {
          sourceType: 'registry',
          packageName: '@platform.skill/pdf-skill',
          version: '1.2.0',
          computedHash: 'abc123',
        },
      },
    });
  });

  it('preserves project lock source metadata for registry removals', async () => {
    await removeCommand(['@platform.skill/pdf-skill'], { yes: true });

    expect(telemetry.track).toHaveBeenCalledWith(
      expect.objectContaining({
        event: 'remove',
        source: '@platform.skill/pdf-skill',
        sourceType: 'registry',
        skills: '@platform.skill/pdf-skill',
      })
    );
  });
});
