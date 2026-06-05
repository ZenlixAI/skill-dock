import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { existsSync, rmSync, mkdirSync, writeFileSync, readdirSync, readFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runCli, runCliWithInput } from './test-utils.js';
import type { InstalledSkillView } from './installed-view.ts';
import { removeCommand } from './remove.ts';

const {
  rmMock,
  listInstalledViewMock,
  removeSkillFromLockMock,
  getSkillFromLockMock,
  getAllLockedSkillsMock,
  readLocalLockMock,
  removeSkillFromLocalLockMock,
  trackMock,
} = vi.hoisted(() => ({
  rmMock: vi.fn(() => Promise.resolve()),
  listInstalledViewMock: vi.fn<() => Promise<InstalledSkillView[]>>(() => Promise.resolve([])),
  removeSkillFromLockMock: vi.fn<(skillName: string) => Promise<boolean>>(() =>
    Promise.resolve(true)
  ),
  getSkillFromLockMock: vi.fn(() => Promise.resolve(null)),
  getAllLockedSkillsMock: vi.fn(() => Promise.resolve({})),
  readLocalLockMock: vi.fn(() => Promise.resolve({ version: 1, skills: {} })),
  removeSkillFromLocalLockMock: vi.fn<(skillName: string, cwd?: string) => Promise<boolean>>(() =>
    Promise.resolve(true)
  ),
  trackMock: vi.fn(),
}));

vi.mock('./detect-agent.ts', () => ({
  detectAgent: vi.fn(() => Promise.resolve({ isAgent: false })),
}));

vi.mock('./agents.ts', () => ({
  agents: {
    cursor: {
      displayName: 'Cursor',
      skillsDir: '.cursor/rules',
    },
    'claude-code': {
      displayName: 'Claude Code',
      skillsDir: '.claude/skills',
    },
  },
  detectInstalledAgents: vi.fn(() => Promise.resolve([])),
}));

vi.mock('./installed-view.ts', () => ({
  listInstalledView: listInstalledViewMock,
}));

vi.mock('./skill-lock.ts', () => ({
  getAllLockedSkills: getAllLockedSkillsMock,
  removeSkillFromLock: removeSkillFromLockMock,
  getSkillFromLock: getSkillFromLockMock,
}));

vi.mock('./local-lock.ts', () => ({
  readLocalLock: readLocalLockMock,
  removeSkillFromLocalLock: removeSkillFromLocalLockMock,
}));

vi.mock('./telemetry.ts', () => ({
  track: trackMock,
}));

vi.mock('fs/promises', async (importOriginal) => {
  const actual = await importOriginal<typeof import('fs/promises')>();
  return {
    ...actual,
    rm: rmMock,
  };
});

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

describe('remove command', { timeout: 30000 }, () => {
  let testDir: string;
  let skillsDir: string;

  beforeEach(() => {
    vi.clearAllMocks();
    testDir = join(tmpdir(), `skills-remove-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });

    // Create .agents/skills directory (canonical location)
    skillsDir = join(testDir, '.agents', 'skills');
    mkdirSync(skillsDir, { recursive: true });
  });

  afterEach(() => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  function createTestSkill(name: string, description?: string) {
    const skillDir = join(skillsDir, name);
    mkdirSync(skillDir, { recursive: true });
    writeFileSync(
      join(skillDir, 'SKILL.md'),
      `---
name: ${name}
description: ${description || `A test skill called ${name}`}
---

# ${name}

This is a test skill.
`
    );
  }

  function createAgentSkillsDir(agentName: string) {
    const agentSkillsDir = join(testDir, agentName, 'skills');
    mkdirSync(agentSkillsDir, { recursive: true });
    return agentSkillsDir;
  }

  function createSymlink(skillName: string, targetDir: string) {
    const skillPath = join(skillsDir, skillName);
    const linkPath = join(targetDir, skillName);
    try {
      // Create relative symlink
      const relativePath = join('..', '..', '.agents', 'skills', skillName);
      const { symlinkSync } = require('fs');
      symlinkSync(relativePath, linkPath);
    } catch {
      // Skip if symlinks aren't supported
    }
  }

  describe('with no skills installed', () => {
    it('should show message when no skills found', () => {
      const result = runCli(['remove', '-y'], testDir);
      expect(result.stdout).toContain('No skills found');
      expect(result.stdout).toContain('to remove');
      expect(result.exitCode).toBe(0);
    });

    it('should show error for non-existent skill name', () => {
      const result = runCli(['remove', 'non-existent-skill', '-y'], testDir);
      expect(result.stdout).toContain('No skills found');
      expect(result.exitCode).toBe(0);
    });
  });

  describe('with skills installed', () => {
    beforeEach(() => {
      createTestSkill('skill-one', 'First test skill');
      createTestSkill('skill-two', 'Second test skill');
      createTestSkill('skill-three', 'Third test skill');

      // Create symlinks in agent directories
      const claudeSkillsDir = createAgentSkillsDir('.claude');
      createSymlink('skill-one', claudeSkillsDir);
      createSymlink('skill-two', claudeSkillsDir);

      const clineSkillsDir = createAgentSkillsDir('.cline');
      createSymlink('skill-one', clineSkillsDir);
      createSymlink('skill-three', clineSkillsDir);
    });

    it('should remove specific skill by name with -y flag', () => {
      const result = runCli(['remove', 'skill-one', '-y'], testDir);

      expect(result.stdout).toContain('Successfully removed');
      expect(result.stdout).toContain('1 skill');

      // Verify skill was removed from canonical location
      expect(existsSync(join(skillsDir, 'skill-one'))).toBe(false);

      // Verify other skills still exist
      expect(existsSync(join(skillsDir, 'skill-two'))).toBe(true);
      expect(existsSync(join(skillsDir, 'skill-three'))).toBe(true);
    });

    it('should remove multiple skills by name', () => {
      const result = runCli(['remove', 'skill-one', 'skill-two', '-y'], testDir);

      expect(result.stdout).toContain('Successfully removed');
      expect(result.stdout).toContain('2 skill');

      expect(existsSync(join(skillsDir, 'skill-one'))).toBe(false);
      expect(existsSync(join(skillsDir, 'skill-two'))).toBe(false);
      expect(existsSync(join(skillsDir, 'skill-three'))).toBe(true);
    });

    it('should remove all skills with --all flag', () => {
      const result = runCli(['remove', '--all', '-y'], testDir);

      expect(result.stdout).toContain('Successfully removed');
      expect(result.stdout).toContain('3 skill');

      // All skills removed
      expect(existsSync(join(skillsDir, 'skill-one'))).toBe(false);
      expect(existsSync(join(skillsDir, 'skill-two'))).toBe(false);
      expect(existsSync(join(skillsDir, 'skill-three'))).toBe(false);
    });

    it('should show error for non-existent skill name when skills exist', () => {
      const result = runCli(['remove', 'non-existent', '-y'], testDir);

      expect(result.stdout).toContain('No matching skills');
      expect(result.exitCode).toBe(0);
    });

    it('should be case-insensitive when matching skill names', () => {
      const result = runCli(['remove', 'SKILL-ONE', '-y'], testDir);

      expect(result.stdout).toContain('Successfully removed');
      expect(existsSync(join(skillsDir, 'skill-one'))).toBe(false);
    });

    it('should remove only the specified skill and leave others', () => {
      runCli(['remove', 'skill-two', '-y'], testDir);

      // skill-two removed
      expect(existsSync(join(skillsDir, 'skill-two'))).toBe(false);

      // Others still exist
      expect(existsSync(join(skillsDir, 'skill-one'))).toBe(true);
      expect(existsSync(join(skillsDir, 'skill-three'))).toBe(true);
    });

    it('should remove a registry-installed skill by package_name', () => {
      const packageDir = join(skillsDir, 'platform.skill-pdf-skill');
      mkdirSync(packageDir, { recursive: true });
      writeFileSync(
        join(packageDir, 'SKILL.md'),
        `---
name: pdf-skill
description: PDF processing workflow
---

# PDF Skill
`
      );
      writeFileSync(
        join(testDir, 'skills-lock.json'),
        JSON.stringify(
          {
            version: 1,
            skills: {
              '@platform.skill/pdf-skill': {
                source: '@platform.skill/pdf-skill',
                sourceType: 'registry',
                version: '1.2.0',
                computedHash: 'abc123',
              },
            },
          },
          null,
          2
        )
      );

      const result = runCli(['remove', '@platform.skill/pdf-skill', '-y'], testDir);
      expect(result.stdout).toContain('Successfully removed');
      expect(existsSync(packageDir)).toBe(false);
    });

    it('should not invent package-name aliases for an install that does not expose them', () => {
      const packageDir = join(skillsDir, 'my-skill');
      mkdirSync(packageDir, { recursive: true });
      writeFileSync(
        join(packageDir, 'SKILL.md'),
        `---
name: my-skill
description: Demo skill
---

# My Skill
`
      );

      const result = runCli(['remove', '@company.skill/my-skill', '-y'], testDir);
      expect(result.stdout).toContain('No matching skills');
      expect(existsSync(packageDir)).toBe(true);
    });

    it('does not delete a tracked lock entry when removing an orphaned conflicting canonical skill', () => {
      const orphanDir = join(skillsDir, 'pdf skill');
      mkdirSync(orphanDir, { recursive: true });
      writeFileSync(
        join(orphanDir, 'SKILL.md'),
        `---
name: pdf skill
description: Untracked conflicting install
---

# pdf skill
`
      );

      const trackedDir = join(skillsDir, 'pdf-skill');
      mkdirSync(trackedDir, { recursive: true });
      writeFileSync(
        join(trackedDir, 'SKILL.md'),
        `---
name: pdf-skill
description: Tracked install
---

# pdf-skill
`
      );

      writeFileSync(
        join(testDir, 'skills-lock.json'),
        JSON.stringify(
          {
            version: 1,
            skills: {
              'pdf-skill': {
                sourceType: 'local',
                installName: 'pdf-skill',
                computedHash: 'tracked-hash',
              },
            },
          },
          null,
          2
        )
      );

      const result = runCli(['remove', 'pdf skill', '-y'], testDir);
      expect(result.stdout).toContain('Successfully removed');
      expect(existsSync(orphanDir)).toBe(false);
      expect(existsSync(trackedDir)).toBe(true);

      const parsedLock = JSON.parse(readFileSync(join(testDir, 'skills-lock.json'), 'utf-8'));
      expect(parsedLock.skills['pdf-skill']).toEqual(
        expect.objectContaining({
          sourceType: 'local',
          installName: 'pdf-skill',
          computedHash: 'tracked-hash',
        })
      );
    });

    it('removes a lock-only orphaned entry when the missing install is the only orphan reason', () => {
      writeFileSync(
        join(testDir, 'skills-lock.json'),
        JSON.stringify(
          {
            version: 1,
            skills: {
              '@platform.skill/pdf-skill': {
                sourceType: 'registry',
                packageName: '@platform.skill/pdf-skill',
                installName: 'pdf-skill',
                version: '1.2.0',
                computedHash: 'tracked-hash',
              },
            },
          },
          null,
          2
        )
      );

      const result = runCli(['remove', '@platform.skill/pdf-skill', '-y'], testDir);
      expect(result.stdout).toContain('Successfully removed');

      const parsedLock = JSON.parse(readFileSync(join(testDir, 'skills-lock.json'), 'utf-8'));
      expect(parsedLock.skills['@platform.skill/pdf-skill']).toBeUndefined();
    });

    it('should list skills to remove before confirmation', () => {
      // Answer 'n' to cancel the confirmation prompt
      const result = runCliWithInput(['remove', 'skill-one', 'skill-two'], 'n', testDir);

      // Should show the skills that will be removed
      expect(result.stdout).toContain('Skills to remove');
      expect(result.stdout).toContain('skill-one');
      expect(result.stdout).toContain('skill-two');
      expect(result.stdout).toContain('uninstall');

      // Skills should NOT be removed since we cancelled
      expect(existsSync(join(skillsDir, 'skill-one'))).toBe(true);
      expect(existsSync(join(skillsDir, 'skill-two'))).toBe(true);
    });
  });

  describe('agent filtering', () => {
    beforeEach(() => {
      createTestSkill('test-skill');
      createAgentSkillsDir('.claude');
      createAgentSkillsDir('.cline');
    });

    it('should show error for invalid agent name', () => {
      const result = runCli(['remove', 'test-skill', '--agent', 'invalid-agent', '-y'], testDir);

      expect(result.stdout).toContain('Invalid agents');
      expect(result.stdout).toContain('invalid-agent');
      expect(result.stdout).toContain('Valid agents');
      expect(result.exitCode).toBe(1);
    });

    it('should accept valid agent names', () => {
      // This should not error on agent validation
      const result = runCli(['remove', 'test-skill', '--agent', 'claude-code', '-y'], testDir);
      expect(result.stdout).not.toContain('Invalid agents');
    });

    it('should accept multiple agent names', () => {
      const result = runCli(
        ['remove', 'test-skill', '--agent', 'claude-code', 'cursor', '-y'],
        testDir
      );
      expect(result.stdout).not.toContain('Invalid agents');
    });
  });

  describe('global flag', () => {
    beforeEach(() => {
      createTestSkill('global-skill');
    });

    it('should accept --global flag without error', () => {
      const result = runCli(['remove', 'global-skill', '--global', '-y'], testDir);
      // Command should run without error (skill may not be found in global scope from test dir)
      expect(result.exitCode).toBe(0);
    });
  });

  describe('command aliases', () => {
    beforeEach(() => {
      createTestSkill('alias-test-skill');
    });

    it('should support "rm" alias', () => {
      const result = runCli(['rm', 'alias-test-skill', '-y'], testDir);
      expect(result.stdout).toContain('Successfully removed');
      expect(result.exitCode).toBe(0);
    });

    it('should reject removed "r" alias', () => {
      const result = runCli(['r', 'alias-test-skill', '-y'], testDir);
      expect(result.stdout).toContain('Unknown command: r');
      expect(result.exitCode).toBe(1);
    });
  });

  describe('edge cases', () => {
    it('should handle skill names with special characters', () => {
      createTestSkill('skill-with-dashes');
      createTestSkill('skill_with_underscores');

      const result = runCli(['remove', 'skill-with-dashes', '-y'], testDir);
      expect(result.stdout).toContain('Successfully removed');
      expect(existsSync(join(skillsDir, 'skill-with-dashes'))).toBe(false);
      expect(existsSync(join(skillsDir, 'skill_with_underscores'))).toBe(true);
    });

    it('should handle removing last remaining skill', () => {
      createTestSkill('last-skill');

      const result = runCli(['remove', 'last-skill', '-y'], testDir);
      expect(result.stdout).toContain('Successfully removed');
      expect(result.stdout).toContain('1 skill');

      // Directory should be empty or removed
      const remaining = readdirSync(skillsDir);
      expect(remaining.length).toBe(0);
    });

    it('should handle directory without SKILL.md file', () => {
      // Create a directory without SKILL.md
      const invalidSkillDir = join(skillsDir, 'invalid-skill');
      mkdirSync(invalidSkillDir, { recursive: true });
      writeFileSync(join(invalidSkillDir, 'README.md'), 'Just a readme');

      createTestSkill('valid-skill');

      const result = runCli(['remove', 'valid-skill', '-y'], testDir);
      expect(result.stdout).toContain('Successfully removed');

      // Invalid directory should still be removed
      expect(existsSync(join(skillsDir, 'invalid-skill'))).toBe(true);
    });
  });

  describe('help and info', () => {
    it('should show help with --help', () => {
      const result = runCli(['remove', '--help'], testDir);
      expect(result.stdout).toContain('Usage');
      expect(result.stdout).toContain('remove');
      expect(result.stdout).toContain('--global');
      expect(result.stdout).toContain('--agent');
      expect(result.stdout).toContain('--yes');
      expect(result.exitCode).toBe(0);
    });

    it('should show help with -h', () => {
      const result = runCli(['remove', '-h'], testDir);
      expect(result.stdout).toContain('Usage');
      expect(result.exitCode).toBe(0);
    });
  });

  describe('option parsing', () => {
    beforeEach(() => {
      createTestSkill('parse-test-skill');
    });

    it('should parse -g as global', () => {
      const result = runCli(['remove', 'parse-test-skill', '-g', '-y'], testDir);
      expect(result.stdout).not.toContain('error');
      expect(result.stdout).not.toContain('unrecognized');
    });

    it('should parse --yes flag', () => {
      const result = runCli(['remove', 'parse-test-skill', '--yes'], testDir);
      expect(result.exitCode).toBe(0);
    });

    it('should parse -a as agent', () => {
      const result = runCli(['remove', 'parse-test-skill', '-a', 'claude-code', '-y'], testDir);
      expect(result.stdout).not.toContain('Invalid agents');
    });

    it('should handle multiple values for --agent', () => {
      const result = runCli(
        ['remove', 'parse-test-skill', '--agent', 'claude-code', 'cursor', '-y'],
        testDir
      );
      expect(result.stdout).not.toContain('Invalid agents');
    });
  });

  describe('installed view selection', () => {
    it('matches requested skills by package name before display-name aliases', async () => {
      const oldCwd = process.cwd();
      process.chdir(testDir);

      const records: InstalledSkillView[] = [
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          installName: 'pdf-skill',
          aliases: ['@platform.skill/pdf-skill', 'pdf-skill', 'pdf-skill-display'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [
            {
              agent: 'cursor',
              path: '/tmp/project/.cursor/rules/pdf-skill',
              kind: 'symlink',
            },
          ],
          sourceType: 'registry',
          version: '1.2.3',
          orphaned: false,
          orphanReasons: [],
        },
        {
          displayName: '@platform.skill/pdf-skill',
          installName: 'pdf-skill-display',
          aliases: ['pdf-skill-display', '@platform.skill/pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill-display',
          agentLinks: [],
          sourceType: 'local',
          orphaned: false,
          orphanReasons: [],
        },
      ];
      listInstalledViewMock.mockResolvedValue(records);

      try {
        await removeCommand(['@platform.skill/pdf-skill'], { yes: true });
      } finally {
        process.chdir(oldCwd);
      }

      expect(rmMock).toHaveBeenCalledWith(
        '/tmp/project/.agents/skills/pdf-skill',
        expect.objectContaining({ recursive: true, force: true })
      );
      expect(rmMock).not.toHaveBeenCalledWith(
        '/tmp/project/.agents/skills/pdf-skill-display',
        expect.anything()
      );
    });

    it('does not match a package-style request when the record only exposes local aliases', async () => {
      const oldCwd = process.cwd();
      process.chdir(testDir);

      const records: InstalledSkillView[] = [
        {
          displayName: 'My Skill',
          installName: 'my-skill',
          aliases: ['my-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/my-skill',
          agentLinks: [],
          sourceType: 'local',
          orphaned: false,
          orphanReasons: [],
        },
      ];
      listInstalledViewMock.mockResolvedValue(records);

      try {
        await removeCommand(['@company.skill/my-skill'], { yes: true });
      } finally {
        process.chdir(oldCwd);
      }

      expect(rmMock).not.toHaveBeenCalled();
      expect(removeSkillFromLocalLockMock).not.toHaveBeenCalled();
    });

    it('does not delete another lock entry when removing an orphaned conflicting record', async () => {
      const oldCwd = process.cwd();
      process.chdir(testDir);

      const records: InstalledSkillView[] = [
        {
          displayName: 'pdf skill',
          installName: 'pdf-skill',
          aliases: ['pdf skill', 'pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf skill',
          agentLinks: [],
          sourceType: 'local',
          orphaned: true,
          orphanReasons: ['missing_lock_entry'],
        },
        {
          displayName: 'pdf-skill',
          installName: 'pdf-skill',
          aliases: ['pdf-skill'],
          scope: 'project',
          canonicalPath: '/tmp/project/.agents/skills/pdf-skill',
          agentLinks: [],
          sourceType: 'local',
          orphaned: false,
          orphanReasons: [],
          lockEntry: {
            sourceType: 'local',
            computedHash: 'tracked-hash',
          },
        },
      ];
      listInstalledViewMock.mockResolvedValue(records);
      removeSkillFromLocalLockMock.mockImplementation(async (skillName) => skillName === 'pdf-skill');

      try {
        await removeCommand(['pdf skill'], { yes: true });
      } finally {
        process.chdir(oldCwd);
      }

      expect(rmMock).toHaveBeenCalledWith(
        '/tmp/project/.agents/skills/pdf skill',
        expect.objectContaining({ recursive: true, force: true })
      );
      expect(removeSkillFromLocalLockMock).not.toHaveBeenCalled();
    });
  });
});
