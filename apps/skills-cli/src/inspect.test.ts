import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { existsSync, rmSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runCli, stripLogo, runCliOutput } from './test-utils.ts';

vi.mock('./packaging-session.ts', () => ({
  runPackagingSession: vi.fn(),
}));

import { runInspect } from './inspect.ts';
import { runPackagingSession } from './packaging-session.ts';

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

  it('should treat a path after missing --registry value as the inspect path', () => {
    const skillDir = join(testDir, 'flag-path-skill');
    mkdirSync(skillDir, { recursive: true });
    mkdirSync(join(skillDir, 'commands'), { recursive: true });
    writeFileSync(
      join(skillDir, 'SKILL.md'),
      `---
name: flag-path-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(skillDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/flag-path-skill', version: '0.2.0' })
    );

    const result = runCli(['inspect', '--registry', skillDir], testDir);

    expect(stripLogo(result.stdout)).toContain('Inspect OK');
    expect(stripLogo(result.stdout)).toContain(`Directory: ${skillDir}`);
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
    expect(output).toContain('Usage: skills inspect [path] [options]');
    expect(output).toContain('Validate a local skill directory');
    expect(output).toContain('--registry <url>');
    expect(output).toContain('--strict');
    expect(output).toContain('skills inspect ./my-skill');
  });
});

describe('inspect command with packaging API', () => {
  let testDir: string;
  let logSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-inspect-api-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: inspect-skill
description: API-backed inspect
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/inspect-skill', version: '1.0.0' }, null, 2)
    );

    logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.mocked(runPackagingSession).mockReset();
    vi.stubEnv('SKILLS_REGISTRY_URL', 'https://catalog.example.com');
    process.exitCode = undefined;
  });

  afterEach(() => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
    logSpy.mockRestore();
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
    process.exitCode = undefined;
  });

  it('uses the packaging session seam when --registry is provided', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      packageName: '@platform.skill/inspect-skill',
      version: '1.0.0',
      displayName: 'inspect-skill',
      description: 'API-backed inspect',
      publishRegistry: undefined,
      components: ['commands'],
      errors: [],
      workspacePath: '/tmp/workspace',
      idempotencyKey: 'packaging-test-key',
      manifestDigest: 'sha256:test',
    });

    await runInspect(['--registry', 'https://flag.example.com', testDir]);

    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      registry: 'https://flag.example.com',
    });
    expect(logSpy).toHaveBeenCalledWith('Inspect OK');
    expect(process.exitCode).toBeUndefined();
  });

  it('passes strict mode through to the packaging session seam', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      packageName: '@platform.skill/inspect-skill',
      version: '1.0.0',
      displayName: 'inspect-skill',
      description: 'API-backed inspect',
      publishRegistry: undefined,
      components: ['commands'],
      errors: [],
      workspacePath: '/tmp/workspace',
      idempotencyKey: 'packaging-test-key',
      manifestDigest: 'sha256:test',
    });

    await runInspect(['--registry', 'https://flag.example.com', '--strict', testDir]);

    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      registry: 'https://flag.example.com',
      strict: true,
    });
  });

  it('uses the packaging session seam when the registry comes from the environment', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      packageName: '@platform.skill/inspect-skill',
      version: '1.0.0',
      displayName: 'inspect-skill',
      description: 'API-backed inspect',
      publishRegistry: undefined,
      components: ['commands'],
      errors: [],
      workspacePath: '/tmp/workspace',
      idempotencyKey: 'packaging-test-key',
      manifestDigest: 'sha256:test',
    });

    await runInspect([testDir]);

    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      registry: 'https://catalog.example.com',
    });
    expect(logSpy).toHaveBeenCalledWith('Inspect OK');
    expect(process.exitCode).toBeUndefined();
  });

  it('prints normalization warnings from registry-backed inspection', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      packageName: '@platform.skill/inspect-skill',
      version: '1.0.0',
      displayName: 'inspect-skill',
      description: 'API-backed inspect',
      publishRegistry: undefined,
      components: ['commands'],
      errors: [],
      workspacePath: '/tmp/workspace',
      idempotencyKey: 'packaging-test-key',
      manifestDigest: 'sha256:test',
      warnings: ['normalized_missing_capability_content'],
      warningDetails: [
        {
          code: 'normalized_missing_capability_content',
          severity: 'warning',
          title: 'Generated placeholder capability content',
          message: 'The imported skill did not contain publishable capability content.',
          suggestedAction: 'Replace the placeholder with real capability content.',
        },
      ],
      importReport: {
        sourceType: 'imported',
        strictMode: false,
        normalized: true,
        generatedPackageJson: false,
        generatedCapabilityPlaceholder: true,
        rewrotePackageIdentity: false,
        generatedPaths: ['docs/README.md'],
        warningCodes: ['normalized_missing_capability_content'],
      },
    });

    await runInspect(['--registry', 'https://flag.example.com', testDir]);

    expect(logSpy).toHaveBeenCalledWith('Import summary:');
    expect(logSpy).toHaveBeenCalledWith('- status: normalized import');
    expect(logSpy).toHaveBeenCalledWith('Warnings:');
    expect(logSpy).toHaveBeenCalledWith('- Generated placeholder capability content');
    expect(logSpy).toHaveBeenCalledWith('  code: normalized_missing_capability_content');
  });

  it('maps a registry-backed failure result to inspect failure output and exit code 1', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: false,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'inspect-skill',
      description: 'API-backed inspect',
      packageName: '@platform.skill/inspect-skill',
      version: '1.0.0',
      publishRegistry: undefined,
      components: ['commands'],
      errors: ['nested_skill_md'],
      idempotencyKey: 'packaging-test-key',
      workspacePath: '/tmp/workspace',
    });

    await runInspect(['--registry', 'https://flag.example.com', testDir]);

    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      registry: 'https://flag.example.com',
    });
    expect(logSpy).toHaveBeenCalledWith('Inspect failed');
    expect(logSpy).toHaveBeenCalledWith('- nested_skill_md');
    expect(process.exitCode).toBe(1);
  });
});
