import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { existsSync, rmSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runCli, runCliOutput, stripLogo } from './test-utils.ts';
import { shouldInstallInternalSkills } from './skills.ts';
import { parseAddOptions } from './add.ts';

describe('add command', () => {
  let testDir: string;

  function writeSkill(
    dir: string,
    name: string,
    description: string,
    metadataBlock?: string
  ): void {
    mkdirSync(dir, { recursive: true });
    writeFileSync(
      join(dir, 'SKILL.md'),
      `---
name: ${name}
description: ${description}
${metadataBlock ? `${metadataBlock}\n` : ''}---
# ${name}
`
    );
  }

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
    expect(result.stdout).toContain('Missing required argument: source');
    expect(result.exitCode).toBe(1);
  });

  it('should list skills from a local path without prompting for agents', () => {
    writeSkill(join(testDir, 'skills', 'zoom-out'), 'zoom-out', 'Broader context');

    const result = runCli(['add', testDir, '--list'], testDir);
    expect(result.stdout).toContain('Found 1 skill');
    expect(result.stdout).toContain('zoom-out');
    expect(result.stdout).not.toContain('Which agents do you want to install to?');
    expect(result.exitCode).toBe(0);
  });

  it('should install from a local path even if the old compatibility env var is set', () => {
    writeSkill(join(testDir, 'zoom-out'), 'zoom-out', 'Broader context');

    const result = runCli(['add', testDir, '--skill', 'zoom-out', '-y'], testDir, {
      SKILLS_ALLOW_LEGACY_ADD: '1',
    });

    expect(result.stdout).toContain('Installation Summary');
    expect(result.stdout).toContain('zoom-out');
    expect(result.exitCode).toBe(0);
  });

  it('should support add command alias (a)', () => {
    const resultA = runCli(['a'], testDir);

    expect(resultA.stdout).toContain('Missing required argument: source');
  });

  it('should reject removed add compatibility aliases', () => {
    const resultI = runCli(['i'], testDir);
    const resultInstall = runCli(['install'], testDir);

    expect(resultI.stdout).toContain('Unknown command: i');
    expect(resultInstall.stdout).toContain('Unknown command: install');
  });

  it('should show dedicated add help output', () => {
    const output = stripLogo(runCliOutput(['add', '--help']));
    expect(output).toContain('Usage: skills add <source> [options]');
    expect(output).toContain('Install a skill from the registry or directly from a source repository/path');
    expect(output).toContain('--version <version>');
    expect(output).toContain('skills add @company.skill/pdf-skill');
    expect(output).toContain('skills add owner/repo --skill pdf');
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
      writeSkill(
        join(testDir, 'internal-skill'),
        'internal-skill',
        'An internal skill',
        'metadata:\n  internal: true'
      );

      const result = runCli(['add', testDir, '--list'], testDir);
      expect(result.stdout).not.toContain('internal-skill');
    });

    it('should show internal skills when INSTALL_INTERNAL_SKILLS=1', () => {
      writeSkill(
        join(testDir, 'internal-skill'),
        'internal-skill',
        'An internal skill',
        'metadata:\n  internal: true'
      );

      const result = runCli(['add', testDir, '--list'], testDir, {
        INSTALL_INTERNAL_SKILLS: '1',
      });
      expect(result.stdout).toContain('internal-skill');
    });

    it('should show internal skills when INSTALL_INTERNAL_SKILLS=true', () => {
      writeSkill(
        join(testDir, 'internal-skill'),
        'internal-skill',
        'An internal skill',
        'metadata:\n  internal: true'
      );

      const result = runCli(['add', testDir, '--list'], testDir, {
        INSTALL_INTERNAL_SKILLS: 'true',
      });
      expect(result.stdout).toContain('internal-skill');
    });

    it('should show non-internal skills alongside internal when env var is set', () => {
      const internalDir = join(testDir, 'skills', 'internal-skill');
      const publicDir = join(testDir, 'skills', 'public-skill');
      writeSkill(internalDir, 'internal-skill', 'An internal skill', 'metadata:\n  internal: true');
      writeSkill(publicDir, 'public-skill', 'A public skill');

      const resultWithout = runCli(['add', testDir, '--list'], testDir);
      expect(resultWithout.stdout).toContain('public-skill');
      expect(resultWithout.stdout).not.toContain('internal-skill');

      const resultWith = runCli(['add', testDir, '--list'], testDir, {
        INSTALL_INTERNAL_SKILLS: '1',
      });
      expect(resultWith.stdout).toContain('public-skill');
      expect(resultWith.stdout).toContain('internal-skill');
    });

    it('should not treat metadata.internal: false as internal', () => {
      writeSkill(
        join(testDir, 'not-internal-skill'),
        'not-internal-skill',
        'Explicitly not internal',
        'metadata:\n  internal: false'
      );

      const result = runCli(['add', testDir, '--list'], testDir);
      expect(result.stdout).toContain('not-internal-skill');
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
