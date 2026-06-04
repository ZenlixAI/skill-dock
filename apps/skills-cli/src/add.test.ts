import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { existsSync, rmSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runCli, runCliOutput, stripLogo } from './test-utils.ts';
import { shouldInstallInternalSkills } from './skills.ts';
import { parseAddOptions } from './add.ts';

describe('add command', () => {
  let testDir: string;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-add-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });
  });

  afterEach(() => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('should show error when no source provided', () => {
    const result = runCli(['add'], testDir);
    expect(result.stdout).toContain('ERROR');
    expect(result.stdout).toContain('Missing required argument: package_name');
    expect(result.exitCode).toBe(1);
  });

  it('should reject source-style identifiers', () => {
    const result = runCli(['add', 'vercel-labs/skills', '-y'], testDir);
    expect(result.stdout).toContain('package_name');
    expect(result.stdout).toContain('owner/repo');
    expect(result.exitCode).toBe(1);
  });

  it('should reject local paths', () => {
    const result = runCli(['add', testDir, '--list'], testDir);
    expect(result.stdout).toContain('package_name');
    expect(result.stdout).toContain('local path');
    expect(result.exitCode).toBe(1);
  });

  it('should reject local paths even if the old compatibility env var is set', () => {
    const result = runCli(['add', testDir, '-y'], testDir, {
      SKILLS_ALLOW_LEGACY_ADD: '1',
    });
    expect(result.stdout).toContain('package_name');
    expect(result.stdout).toContain('local path');
    expect(result.exitCode).toBe(1);
  });

  it('should support add command alias (a)', () => {
    const resultA = runCli(['a'], testDir);

    expect(resultA.stdout).toContain('Missing required argument: package_name');
  });

  it('should reject removed add compatibility aliases', () => {
    const resultI = runCli(['i'], testDir);
    const resultInstall = runCli(['install'], testDir);

    expect(resultI.stdout).toContain('Unknown command: i');
    expect(resultInstall.stdout).toContain('Unknown command: install');
  });

  it('should show dedicated add help output', () => {
    const output = stripLogo(runCliOutput(['add', '--help']));
    expect(output).toContain('Usage: skills add <package_name> [options]');
    expect(output).toContain('Add a skill package from the registry');
    expect(output).toContain('--version <version>');
    expect(output).toContain('skills add @company.skill/pdf-skill');
  });

  it('should parse registry add options', () => {
    const result = parseAddOptions([
      '@platform.skill/pdf-skill',
      '--version',
      '1.2.3',
      '--registry',
      'https://registry.example.com',
      '--verbose',
      '-g',
      '-y',
    ]);

    expect(result.source).toEqual(['@platform.skill/pdf-skill']);
    expect(result.options.version).toBe('1.2.3');
    expect(result.options.registry).toBe('https://registry.example.com');
    expect(result.options.verbose).toBe(true);
    expect(result.options.global).toBe(true);
    expect(result.options.yes).toBe(true);
  });

  it('should reject non-exact version ranges for registry installs', () => {
    const result = runCli(
      ['add', '@platform.skill/pdf-skill', '--version', '^1.2.3', '-y'],
      testDir
    );

    expect(result.stdout).toContain('exact version');
    expect(result.stdout).toContain('--version');
    expect(result.exitCode).toBe(1);
  });

  it('should reject unsupported legacy install flags for registry installs', () => {
    const result = runCli(['add', '@platform.skill/pdf-skill', '--list', '-y'], testDir);

    expect(result.stdout).toContain('Registry installs do not support');
    expect(result.stdout).toContain('--list');
    expect(result.exitCode).toBe(1);
  });

  it('should reject --skill for registry installs', () => {
    const result = runCli(
      ['add', '@platform.skill/pdf-skill', '--skill', 'frontend-design', '-y'],
      testDir
    );

    expect(result.stdout).toContain('Registry installs do not support');
    expect(result.stdout).toContain('--skill');
    expect(result.exitCode).toBe(1);
  });

  it('should reject --all for registry installs', () => {
    const result = runCli(['add', '@platform.skill/pdf-skill', '--all', '-y'], testDir);

    expect(result.stdout).toContain('Registry installs do not support');
    expect(result.stdout).toContain('--all');
    expect(result.exitCode).toBe(1);
  });

  it('should reject --full-depth for registry installs', () => {
    const result = runCli(['add', '@platform.skill/pdf-skill', '--full-depth', '-y'], testDir);

    expect(result.stdout).toContain('Registry installs do not support');
    expect(result.stdout).toContain('--full-depth');
    expect(result.exitCode).toBe(1);
  });

  it('should reject the removed openclaw compatibility flag', () => {
    const result = runCli(
      ['add', '@platform.skill/pdf-skill', '--dangerously-accept-openclaw-risks', '-y'],
      testDir
    );

    expect(result.stdout).toContain('Registry installs do not support');
    expect(result.stdout).toContain('--dangerously-accept-openclaw-risks');
    expect(result.exitCode).toBe(1);
  });

  describe('internal skills', () => {
    it('should skip internal skills by default', () => {
      // Create an internal skill
      const skillDir = join(testDir, 'internal-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: internal-skill
description: An internal skill
metadata:
  internal: true
---

# Internal Skill

This is an internal skill.
`
      );

      const result = runCli(['add', testDir, '--list'], testDir);
      expect(result.stdout).not.toContain('internal-skill');
    });

    it('should show internal skills when INSTALL_INTERNAL_SKILLS=1', () => {
      // Create an internal skill
      const skillDir = join(testDir, 'internal-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: internal-skill
description: An internal skill
metadata:
  internal: true
---

# Internal Skill

This is an internal skill.
`
      );

      const result = runCli(['add', testDir, '--list'], testDir, {
        INSTALL_INTERNAL_SKILLS: '1',
      });
      expect(result.stdout).toContain('package_name');
    });

    it('should show internal skills when INSTALL_INTERNAL_SKILLS=true', () => {
      // Create an internal skill
      const skillDir = join(testDir, 'internal-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: internal-skill
description: An internal skill
metadata:
  internal: true
---

# Internal Skill

This is an internal skill.
`
      );

      const result = runCli(['add', testDir, '--list'], testDir, {
        INSTALL_INTERNAL_SKILLS: 'true',
      });
      expect(result.stdout).toContain('package_name');
    });

    it('should show non-internal skills alongside internal when env var is set', () => {
      // Create both internal and non-internal skills
      const internalDir = join(testDir, 'skills', 'internal-skill');
      const publicDir = join(testDir, 'skills', 'public-skill');
      mkdirSync(internalDir, { recursive: true });
      mkdirSync(publicDir, { recursive: true });

      writeFileSync(
        join(internalDir, 'SKILL.md'),
        `---
name: internal-skill
description: An internal skill
metadata:
  internal: true
---
# Internal Skill
`
      );

      writeFileSync(
        join(publicDir, 'SKILL.md'),
        `---
name: public-skill
description: A public skill
---
# Public Skill
`
      );

      // Without env var - only public skill visible
      const resultWithout = runCli(['add', testDir, '--list'], testDir);
      expect(resultWithout.stdout).toContain('package_name');

      // With env var - add still rejects local paths under the registry-first contract
      const resultWith = runCli(['add', testDir, '--list'], testDir, {
        INSTALL_INTERNAL_SKILLS: '1',
      });
      expect(resultWith.stdout).toContain('package_name');
    });

    it('should not treat metadata.internal: false as internal', () => {
      const skillDir = join(testDir, 'not-internal-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: not-internal-skill
description: Explicitly not internal
metadata:
  internal: false
---
# Not Internal
`
      );

      const result = runCli(['add', testDir, '--list'], testDir);
      expect(result.stdout).toContain('package_name');
    });
  });
});

describe('shouldInstallInternalSkills', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    vi.resetModules();
    process.env = { ...originalEnv };
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it('should return false when INSTALL_INTERNAL_SKILLS is not set', () => {
    delete process.env.INSTALL_INTERNAL_SKILLS;
    expect(shouldInstallInternalSkills()).toBe(false);
  });

  it('should return true when INSTALL_INTERNAL_SKILLS=1', () => {
    process.env.INSTALL_INTERNAL_SKILLS = '1';
    expect(shouldInstallInternalSkills()).toBe(true);
  });

  it('should return true when INSTALL_INTERNAL_SKILLS=true', () => {
    process.env.INSTALL_INTERNAL_SKILLS = 'true';
    expect(shouldInstallInternalSkills()).toBe(true);
  });

  it('should return false for other values', () => {
    process.env.INSTALL_INTERNAL_SKILLS = '0';
    expect(shouldInstallInternalSkills()).toBe(false);

    process.env.INSTALL_INTERNAL_SKILLS = 'false';
    expect(shouldInstallInternalSkills()).toBe(false);

    process.env.INSTALL_INTERNAL_SKILLS = 'yes';
    expect(shouldInstallInternalSkills()).toBe(false);
  });
});

describe('parseAddOptions', () => {
  it('should parse --all flag for legacy rejection', () => {
    const result = parseAddOptions(['@platform.skill/pdf-skill', '--all']);
    expect(result.source).toEqual(['@platform.skill/pdf-skill']);
    expect(result.options.all).toBe(true);
  });

  it('should parse --agent with wildcard', () => {
    const result = parseAddOptions(['@platform.skill/pdf-skill', '--agent', '*']);
    expect(result.source).toEqual(['@platform.skill/pdf-skill']);
    expect(result.options.agent).toEqual(['*']);
  });

  it('should parse --full-depth flag for legacy rejection', () => {
    const result = parseAddOptions(['@platform.skill/pdf-skill', '--full-depth']);
    expect(result.source).toEqual(['@platform.skill/pdf-skill']);
    expect(result.options.fullDepth).toBe(true);
  });

  it('should parse --full-depth with other legacy flags', () => {
    const result = parseAddOptions(['@platform.skill/pdf-skill', '--full-depth', '--list', '-g']);
    expect(result.source).toEqual(['@platform.skill/pdf-skill']);
    expect(result.options.fullDepth).toBe(true);
    expect(result.options.list).toBe(true);
    expect(result.options.global).toBe(true);
  });
});
