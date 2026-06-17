import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./import-git-workflow.ts', () => ({
  runImportGitWorkflow: vi.fn(),
}));

import { parseImportGitOptions, runImportGit } from './import-git.ts';
import { runImportGitWorkflow } from './import-git-workflow.ts';
import { runCliOutput, stripLogo } from './test-utils.ts';

describe('import-git command', () => {
  let logSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.mocked(runImportGitWorkflow).mockReset();
    process.exitCode = undefined;
  });

  afterEach(() => {
    logSpy.mockRestore();
    vi.restoreAllMocks();
    process.exitCode = undefined;
  });

  it('parses explicit import-git options', () => {
    expect(
      parseImportGitOptions([
        'owner/repo',
        '--registry',
        'https://catalog.example.com',
        '--package-name',
        '@acme.skill/legacy-skill',
        '--version',
        '0.1.0',
        '--publish-registry',
        'https://npm.example.com',
        '--skill',
        'pdf',
        '--full-depth',
        '--strict',
      ])
    ).toEqual({
      source: 'owner/repo',
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      publishRegistry: 'https://npm.example.com',
      skill: ['pdf'],
      fullDepth: true,
      strict: true,
    });
  });

  it('prints success output from the import-git workflow result', async () => {
    vi.mocked(runImportGitWorkflow).mockResolvedValueOnce({
      ok: true,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      manifestDigest: 'sha256:test',
      components: ['docs'],
      publishedToRegistry: true,
      created: true,
      idempotentReplay: false,
      versionStatus: 'published',
      installPolicy: 'resolvable',
    });

    await runImportGit([
      'owner/repo',
      '--package-name',
      '@acme.skill/legacy-skill',
      '--version',
      '0.1.0',
    ]);

    expect(runImportGitWorkflow).toHaveBeenCalledWith({
      source: 'owner/repo',
      registry: undefined,
      strict: undefined,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      publishRegistry: undefined,
      skill: undefined,
      fullDepth: undefined,
    });
    expect(logSpy).toHaveBeenCalledWith('Import OK');
    expect(logSpy).toHaveBeenCalledWith('package_name: @acme.skill/legacy-skill');
    expect(logSpy).toHaveBeenCalledWith('version: 0.1.0');
    expect(logSpy).toHaveBeenCalledWith('manifest_digest: sha256:test');
    expect(logSpy).toHaveBeenCalledWith('components: docs');
  });

  it('prints failure output when the workflow fails', async () => {
    vi.mocked(runImportGitWorkflow).mockResolvedValueOnce({
      ok: false,
      errors: ['No skills found in the provided source.'],
    });

    await runImportGit(['owner/repo']);

    expect(logSpy).toHaveBeenCalledWith('Import failed');
    expect(logSpy).toHaveBeenCalledWith('- No skills found in the provided source.');
    expect(process.exitCode).toBe(1);
  });

  it('shows dedicated import-git help output', () => {
    const output = stripLogo(runCliOutput(['import-git', '--help']));
    expect(output).toContain('Usage: skills import-git <source> [options]');
    expect(output).toContain(
      'Clone an external git source, discover a skill, and publish it through Catalog one-shot import.'
    );
    expect(output).toContain('--skill <name>');
    expect(output).toContain('--full-depth');
  });
});
