import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { existsSync, rmSync, readFileSync, mkdirSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runCliOutput, stripLogo } from './test-utils.ts';

describe('init command', () => {
  let testDir: string;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });
  });

  afterEach(() => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('should initialize a skill and create SKILL.md', () => {
    const output = stripLogo(runCliOutput(['init', 'my-test-skill'], testDir));
    expect(output).toMatchInlineSnapshot(`
      "Initialized skill: my-test-skill

      Created:
        my-test-skill/SKILL.md
        my-test-skill/package.json
        my-test-skill/commands/

      Next:
        - Edit my-test-skill/SKILL.md
        - Run skills inspect
        - Run skills publish

      "
    `);

    const skillPath = join(testDir, 'my-test-skill', 'SKILL.md');
    const packageJsonPath = join(testDir, 'my-test-skill', 'package.json');
    const commandsPath = join(testDir, 'my-test-skill', 'commands');
    expect(existsSync(skillPath)).toBe(true);
    expect(existsSync(packageJsonPath)).toBe(true);
    expect(existsSync(commandsPath)).toBe(true);

    const content = readFileSync(skillPath, 'utf-8');
    expect(content).toMatchInlineSnapshot(`
      "---
      name: my-test-skill
      description: A brief description of what this skill does
      ---

      # my-test-skill

      Instructions for the agent to follow when this skill is activated.

      ## When to use

      Describe when this skill should be used.

      ## Instructions

      1. First step
      2. Second step
      3. Additional steps as needed
      "
    `);

    const packageJson = JSON.parse(readFileSync(packageJsonPath, 'utf-8'));
    expect(packageJson).toEqual({
      name: '@company.skill/my-test-skill',
      version: '0.1.0',
    });
  });

  it('should allow multiple skills in same directory', () => {
    runCliOutput(['init', 'hydration-fix'], testDir);
    runCliOutput(['init', 'waterfall-data-fetching'], testDir);

    expect(existsSync(join(testDir, 'hydration-fix', 'SKILL.md'))).toBe(true);
    expect(existsSync(join(testDir, 'waterfall-data-fetching', 'SKILL.md'))).toBe(true);
  });

  it('should init SKILL.md in cwd when no name provided', () => {
    const output = stripLogo(runCliOutput(['init'], testDir));

    expect(output).toContain('Initialized skill:');
    expect(output).toContain('Created:\n  SKILL.md'); // directly in cwd, not in a subfolder
    expect(output).toContain('package.json');
    expect(output).toContain('Next:');
    expect(output).toContain('Run skills inspect');
    expect(output).toContain('Run skills publish');
    expect(output).not.toContain('npx skills');
    expect(existsSync(join(testDir, 'SKILL.md'))).toBe(true);
    expect(existsSync(join(testDir, 'package.json'))).toBe(true);
  });

  it('should show concise next steps with skill path', () => {
    const output = stripLogo(runCliOutput(['init', 'my-skill'], testDir));

    expect(output).toContain('Next:');
    expect(output).toContain('- Edit my-skill/SKILL.md');
    expect(output).toContain('- Run skills inspect');
    expect(output).toContain('- Run skills publish');
    expect(output).not.toContain('npx skills');
  });

  it('should show error if skill already exists', () => {
    runCliOutput(['init', 'existing-skill'], testDir);
    const output = stripLogo(runCliOutput(['init', 'existing-skill'], testDir));
    expect(output).toMatchInlineSnapshot(`
      "Skill already exists at existing-skill/SKILL.md
      "
    `);
  });

  it('should show dedicated init help output', () => {
    const output = stripLogo(runCliOutput(['init', '--help'], testDir));
    expect(output).toContain('Usage: skills init [name]');
    expect(output).toContain('Initialize a skill scaffold');
    expect(output).toContain('skills init my-skill');
    expect(output).toContain('@company.skill/my-skill');
  });
});
