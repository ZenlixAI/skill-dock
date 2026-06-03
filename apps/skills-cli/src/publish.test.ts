import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest';

vi.mock('./publish-workflow.ts', () => ({
  runPublishWorkflow: vi.fn(),
}));

import { parsePublishOptions, runPublish } from './publish.ts';
import { runPublishWorkflow } from './publish-workflow.ts';
import { runCliOutput, stripLogo } from './test-utils.ts';

describe('publish command', () => {
  let logSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.mocked(runPublishWorkflow).mockReset();
    process.exitCode = undefined;
  });

  afterEach(() => {
    logSpy.mockRestore();
    vi.restoreAllMocks();
    process.exitCode = undefined;
  });

  it('parses an optional path and registry override', () => {
    expect(parsePublishOptions(['./skill', '--registry', 'https://catalog.example.com'])).toEqual({
      path: './skill',
      registry: 'https://catalog.example.com',
    });
  });

  it('parses strict packaging mode', () => {
    expect(parsePublishOptions(['./skill', '--strict'])).toEqual({
      path: './skill',
      strict: true,
    });
  });

  it('prints success output from the publish workflow result', async () => {
    vi.mocked(runPublishWorkflow).mockResolvedValueOnce({
      ok: true,
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      bundlePath: '/tmp/packaging/workspace/platform.skill-my-skill-1.2.0.tgz',
    });

    await runPublish(['./skill', '--registry', 'https://catalog.example.com']);

    expect(runPublishWorkflow).toHaveBeenCalledWith({
      path: './skill',
      registry: 'https://catalog.example.com',
    });
    expect(logSpy).toHaveBeenCalledWith('Publish OK');
    expect(logSpy).toHaveBeenCalledWith('package_name: @platform.skill/my-skill');
    expect(logSpy).toHaveBeenCalledWith('version: 1.2.0');
    expect(logSpy).toHaveBeenCalledWith(
      'bundle: /tmp/packaging/workspace/platform.skill-my-skill-1.2.0.tgz'
    );
    expect(process.exitCode).toBeUndefined();
  });

  it('prints warnings from the publish workflow result', async () => {
    vi.mocked(runPublishWorkflow).mockResolvedValueOnce({
      ok: true,
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      bundlePath: '/tmp/packaging/workspace/platform.skill-my-skill-1.2.0.tgz',
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

    await runPublish(['./skill', '--strict']);

    expect(runPublishWorkflow).toHaveBeenCalledWith({
      path: './skill',
      registry: undefined,
      strict: true,
    });
    expect(logSpy).toHaveBeenCalledWith('Import summary:');
    expect(logSpy).toHaveBeenCalledWith('- status: normalized import');
    expect(logSpy).toHaveBeenCalledWith('Warnings:');
    expect(logSpy).toHaveBeenCalledWith('- Generated placeholder capability content');
    expect(logSpy).toHaveBeenCalledWith('  code: normalized_missing_capability_content');
    expect(logSpy).toHaveBeenCalledWith(
      '  The imported skill did not contain publishable capability content.'
    );
    expect(logSpy).toHaveBeenCalledWith(
      '  Suggested action: Replace the placeholder with real capability content.'
    );
  });

  it('prints failure output from the publish workflow result', async () => {
    vi.mocked(runPublishWorkflow).mockResolvedValueOnce({
      ok: false,
      errors: ['package.json version is required'],
    });

    await runPublish(['./skill']);

    expect(runPublishWorkflow).toHaveBeenCalledWith({
      path: './skill',
      registry: undefined,
    });
    expect(logSpy).toHaveBeenCalledWith('Publish failed');
    expect(logSpy).toHaveBeenCalledWith('- package.json version is required');
    expect(process.exitCode).toBe(1);
  });

  it('shows dedicated publish help output', () => {
    const output = stripLogo(runCliOutput(['publish', '--help']));
    expect(output).toContain('Usage: skills publish [path] [options]');
    expect(output).toContain('Validate, pack, and publish a local skill directory');
    expect(output).toContain('--registry <url>');
    expect(output).toContain('--strict');
    expect(output).toContain('skills publish ./my-skill');
  });
});
