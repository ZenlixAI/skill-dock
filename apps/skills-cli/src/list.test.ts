import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { existsSync, rmSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runCli, runCliOutput, stripLogo } from './test-utils.ts';
import { parseListOptions } from './list.ts';

describe('list command', () => {
  let testDir: string;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-list-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });
  });

  afterEach(() => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  describe('parseListOptions', () => {
    it('should parse empty args', () => {
      const options = parseListOptions([]);
      expect(options).toEqual({});
    });

    it('should parse -g flag', () => {
      const options = parseListOptions(['-g']);
      expect(options.global).toBe(true);
    });

    it('should parse --global flag', () => {
      const options = parseListOptions(['--global']);
      expect(options.global).toBe(true);
    });

    it('should parse -a flag with single agent', () => {
      const options = parseListOptions(['-a', 'claude-code']);
      expect(options.agent).toEqual(['claude-code']);
    });

    it('should parse --agent flag with single agent', () => {
      const options = parseListOptions(['--agent', 'cursor']);
      expect(options.agent).toEqual(['cursor']);
    });

    it('should parse -a flag with multiple agents', () => {
      const options = parseListOptions(['-a', 'claude-code', 'cursor', 'codex']);
      expect(options.agent).toEqual(['claude-code', 'cursor', 'codex']);
    });

    it('should parse combined flags', () => {
      const options = parseListOptions(['-g', '-a', 'claude-code', 'cursor']);
      expect(options.global).toBe(true);
      expect(options.agent).toEqual(['claude-code', 'cursor']);
    });

    it('should parse --json flag', () => {
      const options = parseListOptions(['--json']);
      expect(options.json).toBe(true);
    });

    it('should parse -p flag', () => {
      const options = parseListOptions(['-p']);
      expect(options.project).toBe(true);
    });

    it('should parse --all-scopes flag', () => {
      const options = parseListOptions(['--all-scopes']);
      expect(options.allScopes).toBe(true);
    });

    it('should parse combined --json and -g flags', () => {
      const options = parseListOptions(['-g', '--json']);
      expect(options.global).toBe(true);
      expect(options.json).toBe(true);
    });

    it('should stop collecting agents at next flag', () => {
      const options = parseListOptions(['-a', 'claude-code', '-g']);
      expect(options.agent).toEqual(['claude-code']);
      expect(options.global).toBe(true);
    });
  });

  describe('CLI integration', () => {
    it('should run list command', () => {
      const result = runCli(['list'], testDir);
      // Empty project dir shows "No project skills found"
      expect(result.stdout).toContain('No project skills found.');
      expect(result.exitCode).toBe(0);
    });

    it('should run ls alias', () => {
      const result = runCli(['ls'], testDir);
      expect(result.stdout).toContain('No project skills found.');
      expect(result.exitCode).toBe(0);
    });

    it('should output empty JSON array when no skills', () => {
      const result = runCli(['list', '--json'], testDir);
      expect(result.exitCode).toBe(0);
      const parsed = JSON.parse(result.stdout.trim());
      expect(parsed).toEqual([]);
    });

    it('should output valid JSON with --json flag', () => {
      const skillDir = join(testDir, '.agents', 'skills', 'json-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: json-skill
description: A skill for JSON testing
---

# JSON Skill
`
      );

      const result = runCli(['list', '--json'], testDir);
      expect(result.exitCode).toBe(0);
      const parsed = JSON.parse(result.stdout.trim());
      expect(Array.isArray(parsed)).toBe(true);
      expect(parsed.length).toBe(1);
      expect(parsed[0].name).toBe('json-skill');
      expect(parsed[0].version).toBeNull();
      expect(parsed[0].path).toContain('json-skill');
      expect(parsed[0].scope).toBe('project');
      expect(Array.isArray(parsed[0].agents)).toBe(true);
      // No ANSI codes in JSON output
      expect(result.stdout).not.toMatch(/\x1b\[/);
    });

    it('should output multiple skills as JSON array', () => {
      const skill1Dir = join(testDir, '.agents', 'skills', 'skill-alpha');
      const skill2Dir = join(testDir, '.agents', 'skills', 'skill-beta');
      mkdirSync(skill1Dir, { recursive: true });
      mkdirSync(skill2Dir, { recursive: true });

      writeFileSync(
        join(skill1Dir, 'SKILL.md'),
        `---\nname: skill-alpha\ndescription: Alpha\n---\n# Alpha\n`
      );
      writeFileSync(
        join(skill2Dir, 'SKILL.md'),
        `---\nname: skill-beta\ndescription: Beta\n---\n# Beta\n`
      );

      const result = runCli(['list', '--json'], testDir);
      expect(result.exitCode).toBe(0);
      const parsed = JSON.parse(result.stdout.trim());
      expect(parsed.length).toBe(2);
      const names = parsed.map((s: any) => s.name);
      expect(names).toContain('skill-alpha');
      expect(names).toContain('skill-beta');
    });

    it('should show message when no project skills found', () => {
      const result = runCli(['list'], testDir);
      expect(result.stdout).toContain('No project skills found.');
      expect(result.stdout).toContain('Try listing global skills with -g.');
      expect(result.exitCode).toBe(0);
    });

    it('should list project skills', () => {
      // Create a skill in the canonical location
      const skillDir = join(testDir, '.agents', 'skills', 'test-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: test-skill
description: A test skill for listing
---

# Test Skill

This is a test skill.
`
      );

      const result = runCli(['list'], testDir);
      expect(result.stdout).toContain('test-skill');
      expect(result.stdout).toContain('Project Skills');
      // Description should not be shown
      expect(result.stdout).not.toContain('A test skill for listing');
      expect(result.exitCode).toBe(0);
    });

    it('should prefer package_name and version from the local lock view', () => {
      const skillDir = join(testDir, '.agents', 'skills', 'platform.skill-pdf-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
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
                sourceType: 'registry',
                packageName: '@platform.skill/pdf-skill',
                version: '1.2.0',
                computedHash: 'abc123',
              },
            },
          },
          null,
          2
        )
      );

      const result = runCli(['list'], testDir);
      expect(result.stdout).toContain('@platform.skill/pdf-skill');
      expect(result.stdout).toContain('v1.2.0');

      const jsonResult = runCli(['list', '--json'], testDir);
      const parsed = JSON.parse(jsonResult.stdout.trim());
      expect(parsed[0].name).toBe('@platform.skill/pdf-skill');
      expect(parsed[0].version).toBe('1.2.0');
    });

    it('should list multiple skills', () => {
      // Create multiple skills
      const skill1Dir = join(testDir, '.agents', 'skills', 'skill-one');
      const skill2Dir = join(testDir, '.agents', 'skills', 'skill-two');
      mkdirSync(skill1Dir, { recursive: true });
      mkdirSync(skill2Dir, { recursive: true });

      writeFileSync(
        join(skill1Dir, 'SKILL.md'),
        `---
name: skill-one
description: First skill
---
# Skill One
`
      );

      writeFileSync(
        join(skill2Dir, 'SKILL.md'),
        `---
name: skill-two
description: Second skill
---
# Skill Two
`
      );

      const result = runCli(['list'], testDir);
      expect(result.stdout).toContain('skill-one');
      expect(result.stdout).toContain('skill-two');
      expect(result.stdout).toContain('Project Skills');
      expect(result.exitCode).toBe(0);
    });

    it('marks orphaned installed skills in human-readable output', () => {
      const skillDir = join(testDir, '.agents', 'skills', 'manual-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: Manual Skill
description: Installed outside the lock
---
# Manual Skill
`
      );

      const result = runCli(['list'], testDir);
      expect(result.stdout).toContain('Manual Skill');
      expect(result.stdout).toContain('(orphaned)');
      expect(result.exitCode).toBe(0);
    });

    it('should respect -g flag for global only', () => {
      const fakeHome = join(testDir, 'fake-home');
      const globalSkillDir = join(fakeHome, '.claude', 'skills', 'global-skill');
      mkdirSync(globalSkillDir, { recursive: true });
      writeFileSync(
        join(globalSkillDir, 'SKILL.md'),
        `---
name: global-skill
description: A global skill
---
# Global Skill
`
      );

      // Create a project skill (should not be shown with -g)
      const skillDir = join(testDir, '.agents', 'skills', 'project-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: project-skill
description: A project skill
---
# Project Skill
`
      );

      const result = runCli(['list', '-g'], testDir, { HOME: fakeHome });
      // Should not show project skill when -g is specified
      expect(result.stdout).not.toContain('project-skill');
      expect(result.stdout).toContain('global-skill');
      expect(result.stdout).toContain('Global Skills');
    });

    it('should show error for invalid agent filter', () => {
      const result = runCli(['list', '-a', 'invalid-agent'], testDir);
      expect(result.stdout).toContain('Invalid agents');
      expect(result.stdout).toContain('invalid-agent');
      expect(result.exitCode).toBe(1);
    });

    it('should filter by valid agent', () => {
      // Create a skill
      const skillDir = join(testDir, '.agents', 'skills', 'test-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: test-skill
description: A test skill
---
# Test Skill
`
      );

      const result = runCli(['list', '-a', 'claude-code'], testDir);
      expect(result.stdout).toContain('test-skill');
      expect(result.exitCode).toBe(0);
    });

    it('should ignore directories without SKILL.md', () => {
      // Create a valid skill
      const validDir = join(testDir, '.agents', 'skills', 'valid-skill');
      mkdirSync(validDir, { recursive: true });
      writeFileSync(
        join(validDir, 'SKILL.md'),
        `---
name: valid-skill
description: Valid skill
---
# Valid
`
      );

      // Create an invalid directory (no SKILL.md)
      const invalidDir = join(testDir, '.agents', 'skills', 'invalid-skill');
      mkdirSync(invalidDir, { recursive: true });
      writeFileSync(join(invalidDir, 'README.md'), '# Not a skill');

      const result = runCli(['list'], testDir);
      expect(result.stdout).toContain('valid-skill');
      expect(result.stdout).not.toContain('invalid-skill');
      expect(result.exitCode).toBe(0);
    });

    it('should handle SKILL.md with missing frontmatter', () => {
      // Create a valid skill
      const validDir = join(testDir, '.agents', 'skills', 'valid-skill');
      mkdirSync(validDir, { recursive: true });
      writeFileSync(
        join(validDir, 'SKILL.md'),
        `---
name: valid-skill
description: Valid skill
---
# Valid
`
      );

      // Create a skill with invalid SKILL.md (no frontmatter)
      const invalidDir = join(testDir, '.agents', 'skills', 'invalid-skill');
      mkdirSync(invalidDir, { recursive: true });
      writeFileSync(join(invalidDir, 'SKILL.md'), '# Invalid\nNo frontmatter here');

      const result = runCli(['list'], testDir);
      expect(result.stdout).toContain('valid-skill');
      expect(result.stdout).not.toContain('invalid-skill');
      expect(result.exitCode).toBe(0);
    });

    it('should show skill path', () => {
      const skillDir = join(testDir, '.agents', 'skills', 'test-skill');
      mkdirSync(skillDir, { recursive: true });
      writeFileSync(
        join(skillDir, 'SKILL.md'),
        `---
name: test-skill
description: A test skill
---
# Test Skill
`
      );

      const result = runCli(['list'], testDir);
      // Path is shown inline with skill name (handles both Unix / and Windows \)
      expect(result.stdout).toMatch(/\.agents[/\\]skills[/\\]test-skill/);
    });

    it('should show both scopes and winner with --all-scopes', () => {
      const projectDir = join(testDir, '.agents', 'skills', 'platform.skill-pdf-skill');
      mkdirSync(projectDir, { recursive: true });
      writeFileSync(
        join(projectDir, 'SKILL.md'),
        `---
name: pdf-skill
description: PDF processing workflow
---
# PDF Skill
`
      );

      const globalBase = join(testDir, 'fake-home');
      const globalDir = join(globalBase, '.agents', 'skills', 'platform.skill-pdf-skill');
      mkdirSync(globalDir, { recursive: true });
      writeFileSync(
        join(globalDir, 'SKILL.md'),
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

      const stateDir = join(globalBase, 'skills');
      mkdirSync(stateDir, { recursive: true });
      writeFileSync(
        join(stateDir, '.skill-lock.json'),
        JSON.stringify(
          {
            version: 3,
            skills: {
              '@platform.skill/pdf-skill': {
                sourceType: 'registry',
                version: '1.1.0',
                packageName: '@platform.skill/pdf-skill',
                skillFolderHash: '',
                installedAt: '',
                updatedAt: '',
              },
            },
          },
          null,
          2
        )
      );

      const result = runCli(['list', '--all-scopes'], testDir, {
        HOME: globalBase,
        XDG_STATE_HOME: globalBase,
      });

      expect(result.stdout).toContain('Installed Skills');
      expect(result.stdout).toContain('@platform.skill/pdf-skill');
      expect(result.stdout).toContain('winner: project');
      expect(result.stdout).toContain('project');
      expect(result.stdout).toContain('global');

      const jsonResult = runCli(['list', '--all-scopes', '--json'], testDir, {
        HOME: globalBase,
        XDG_STATE_HOME: globalBase,
      });
      const parsed = JSON.parse(jsonResult.stdout.trim());
      expect(parsed).toEqual([
        {
          name: '@platform.skill/pdf-skill',
          winner: 'project',
          scopes: [
            {
              scope: 'global',
              version: '1.1.0',
              path: expect.stringContaining('.agents/skills/platform.skill-pdf-skill'),
              agents: expect.any(Array),
              orphaned: false,
            },
            {
              scope: 'project',
              version: '1.2.0',
              path: expect.stringContaining('.agents/skills/platform.skill-pdf-skill'),
              agents: expect.any(Array),
              orphaned: false,
            },
          ],
        },
      ]);
      expect(parsed[0].scopes[0]).not.toHaveProperty('name');
      expect(parsed[0].scopes[0]).not.toHaveProperty('aliases');
      expect(parsed[0].scopes[1]).not.toHaveProperty('name');
      expect(parsed[0].scopes[1]).not.toHaveProperty('aliases');
    });

    it('keeps shared display aliases in separate all-scope groups', () => {
      const projectDir = join(testDir, '.agents', 'skills', 'review-command');
      mkdirSync(projectDir, { recursive: true });
      writeFileSync(
        join(projectDir, 'SKILL.md'),
        `---
name: Review
description: Project review command
---
# Review Command
`
      );

      writeFileSync(
        join(testDir, 'skills-lock.json'),
        JSON.stringify(
          {
            version: 1,
            skills: {
              '@platform.skill/review-command': {
                sourceType: 'registry',
                packageName: '@platform.skill/review-command',
                installName: 'review-command',
                version: '1.0.0',
                computedHash: 'project-hash',
              },
            },
          },
          null,
          2
        )
      );

      const globalBase = join(testDir, 'fake-home');
      const globalDir = join(globalBase, '.agents', 'skills', 'review-helper');
      mkdirSync(globalDir, { recursive: true });
      writeFileSync(
        join(globalDir, 'SKILL.md'),
        `---
name: Review
description: Global review helper
---
# Review Helper
`
      );

      const stateDir = join(globalBase, 'skills');
      mkdirSync(stateDir, { recursive: true });
      writeFileSync(
        join(stateDir, '.skill-lock.json'),
        JSON.stringify(
          {
            version: 3,
            skills: {
              '@platform.skill/review-helper': {
                sourceType: 'registry',
                version: '2.0.0',
                packageName: '@platform.skill/review-helper',
                installName: 'review-helper',
                skillFolderHash: '',
                installedAt: '',
                updatedAt: '',
              },
            },
          },
          null,
          2
        )
      );

      const jsonResult = runCli(['list', '--all-scopes', '--json'], testDir, {
        HOME: globalBase,
        XDG_STATE_HOME: globalBase,
      });
      const parsed = JSON.parse(jsonResult.stdout.trim());

      expect(parsed).toHaveLength(2);
      expect(parsed).toEqual(
        expect.arrayContaining([
          {
            name: '@platform.skill/review-command',
            winner: 'project',
            scopes: [
              {
                scope: 'project',
                version: '1.0.0',
                path: expect.stringContaining('.agents/skills/review-command'),
                agents: expect.any(Array),
                orphaned: false,
              },
            ],
          },
          {
            name: '@platform.skill/review-helper',
            winner: 'global',
            scopes: [
              {
                scope: 'global',
                version: '2.0.0',
                path: expect.stringContaining('.agents/skills/review-helper'),
                agents: expect.any(Array),
                orphaned: false,
              },
            ],
          },
        ])
      );
    });

    it('keeps same installName records separate across scopes when package identity is missing', () => {
      const projectDir = join(testDir, '.agents', 'skills', 'review');
      mkdirSync(projectDir, { recursive: true });
      writeFileSync(
        join(projectDir, 'SKILL.md'),
        `---
name: Project Review
description: Project-local review skill
---
# Project Review
`
      );

      const globalBase = join(testDir, 'fake-home');
      const globalDir = join(globalBase, '.agents', 'skills', 'review');
      mkdirSync(globalDir, { recursive: true });
      writeFileSync(
        join(globalDir, 'SKILL.md'),
        `---
name: Global Review
description: Global-local review skill
---
# Global Review
`
      );

      const jsonResult = runCli(['list', '--all-scopes', '--json'], testDir, {
        HOME: globalBase,
        XDG_STATE_HOME: globalBase,
      });
      const parsed = JSON.parse(jsonResult.stdout.trim());

      expect(parsed).toHaveLength(2);
      expect(parsed).toEqual(
        expect.arrayContaining([
          {
            name: 'Project Review',
            winner: 'project',
            scopes: [
              {
                scope: 'project',
                version: null,
                path: expect.stringContaining('.agents/skills/review'),
                agents: expect.any(Array),
                orphaned: true,
              },
            ],
          },
          {
            name: 'Global Review',
            winner: 'global',
            scopes: [
              {
                scope: 'global',
                version: null,
                path: expect.stringContaining('.agents/skills/review'),
                agents: expect.any(Array),
                orphaned: true,
              },
            ],
          },
        ])
      );
    });
  });

  describe('help output', () => {
    it('should show dedicated list help output', () => {
      const output = stripLogo(runCliOutput(['list', '--help']));
      expect(output).toContain('Usage: skills list [options]');
      expect(output).toContain('List installed skills');
      expect(output).toContain('--all-scopes');
      expect(output).toContain('skills ls -g');
    });

    it('should include list command in help', () => {
      const result = runCli(['--help']);
      expect(result.stdout).toContain('list, ls');
      expect(result.stdout).toContain('List installed skills');
    });

    it('should include list options in help', () => {
      const result = runCli(['--help']);
      expect(result.stdout).toContain('List Options:');
      expect(result.stdout).toContain('-g, --global');
      expect(result.stdout).toContain('-p, --project');
      expect(result.stdout).toContain('-a, --agent');
    });

    it('should include list examples in help', () => {
      const result = runCli(['--help']);
      expect(result.stdout).toContain('skills list');
      expect(result.stdout).toContain('skills ls -g');
      expect(result.stdout).toContain('skills ls -a claude-code');
    });
  });

  describe('default output', () => {
    it('should include list command in the concise help output', () => {
      const result = runCli([]);
      expect(result.stdout).toContain('Usage: skills <command> [options]');
      expect(result.stdout).toContain('list, ls');
      expect(result.stdout).toContain('List installed skills');
      expect(result.stdout).not.toContain('The open agent skills ecosystem');
    });
  });
});
