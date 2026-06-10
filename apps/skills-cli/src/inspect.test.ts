import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { existsSync, rmSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runCli, stripLogo, runCliOutput } from './test-utils.ts';

describe('inspect command', () => {
  let testDir: string;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-inspect-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });
    vi.stubEnv('SKILLS_REGISTRY_URL', '');
  });

  afterEach(() => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
    vi.unstubAllEnvs();
  });

  it('should validate a skill directory with SKILL.md and package.json', () => {
    mkdirSync(join(testDir, 'commands'), { recursive: true });
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---

# My Skill
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/my-skill', version: '1.2.0' }, null, 2)
    );

    const result = runCli(['inspect'], testDir);

    expect(stripLogo(result.stdout)).toContain('Inspect OK');
    expect(stripLogo(result.stdout)).toContain('SKILL.md');
    expect(stripLogo(result.stdout)).toContain('package.json');
    expect(result.exitCode).toBe(0);
  });

  it('should inspect a provided path', () => {
    const skillDir = join(testDir, 'custom-skill');
    mkdirSync(skillDir, { recursive: true });
    mkdirSync(join(skillDir, 'commands'), { recursive: true });
    writeFileSync(
      join(skillDir, 'SKILL.md'),
      `---
name: custom-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(skillDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/custom-skill', version: '0.1.0' })
    );

    const result = runCli(['inspect', skillDir], testDir);

    expect(stripLogo(result.stdout)).toContain('Inspect OK');
    expect(result.exitCode).toBe(0);
  });

  it('should fail when package.json is missing required fields', () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: broken-skill
description: Missing version
---
`
    );
    writeFileSync(join(testDir, 'package.json'), JSON.stringify({ name: 'broken-skill' }, null, 2));

    const result = runCli(['inspect'], testDir);

    expect(stripLogo(result.stdout || result.stderr)).toContain('Inspect failed');
    expect(stripLogo(result.stdout || result.stderr)).toContain('package.json version is required');
    expect(result.exitCode).toBe(1);
  });

  it('should fail local inspect when the skill contains a nested SKILL.md', () => {
    mkdirSync(join(testDir, 'commands', 'nested'), { recursive: true });
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: nested-root
description: Root skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/nested-root', version: '1.2.0' }, null, 2)
    );
    writeFileSync(
      join(testDir, 'commands', 'nested', 'SKILL.md'),
      `---
name: nested-child
description: Should not exist
---
`
    );

    const result = runCli(['inspect'], testDir);

    expect(stripLogo(result.stdout || result.stderr)).toContain('Inspect failed');
    expect(stripLogo(result.stdout || result.stderr)).toContain('nested_skill_md');
    expect(result.exitCode).toBe(1);
  });

  it('should show inspect in the main help output', () => {
    const output = stripLogo(runCliOutput(['--help']));
    expect(output).toContain('inspect [path]');
  });

  it('should show dedicated inspect help output', () => {
    const output = stripLogo(runCliOutput(['inspect', '--help']));
    expect(output).toContain('Usage: skills inspect [path]');
    expect(output).toContain('Validate a local skill directory');
    expect(output).toContain('skills inspect ./my-skill');
  });
 
  it('should ignore SKILLS_REGISTRY_URL and remain local-only', () => {
    vi.stubEnv('SKILLS_REGISTRY_URL', 'https://catalog.example.com');
    mkdirSync(join(testDir, 'commands'), { recursive: true });
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: local-only-skill
description: Should stay local
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/local-only-skill', version: '0.3.0' }, null, 2)
    );

    const result = runCli(['inspect'], testDir);

    expect(stripLogo(result.stdout)).toContain('Inspect OK');
    expect(stripLogo(result.stdout)).not.toContain('workspace_handle:');
    expect(stripLogo(result.stdout)).not.toContain('Import summary:');
    expect(stripLogo(result.stdout)).not.toContain('Warnings:');
    expect(result.exitCode).toBe(0);
  });
});
