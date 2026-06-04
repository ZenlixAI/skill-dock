import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest';

vi.mock('./import-local-workflow.ts', () => ({
  runImportLocalWorkflow: vi.fn(),
}));

import { parseImportLocalOptions, runImportLocal } from './import-local.ts';
import { runImportLocalWorkflow } from './import-local-workflow.ts';
import { runCliOutput, stripLogo } from './test-utils.ts';

describe('import-local command', () => {
  let logSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.mocked(runImportLocalWorkflow).mockReset();
    process.exitCode = undefined;
  });

  afterEach(() => {
    logSpy.mockRestore();
    vi.restoreAllMocks();
    process.exitCode = undefined;
  });

  it('parses explicit import-local options', () => {
    expect(
      parseImportLocalOptions([
        './legacy-skill',
        '--registry',
        'https://catalog.example.com',
        '--package-name',
        '@acme.skill/legacy-skill',
        '--version',
        '0.1.0',
        '--publish-registry',
        'https://npm.example.com',
        '--npm-registry',
        'https://npm.example.com',
        '--strict',
      ])
    ).toEqual({
      path: './legacy-skill',
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      publishRegistry: 'https://npm.example.com',
      npmRegistry: 'https://npm.example.com',
      strict: true,
    });
  });

  it('prints success output from the import-local workflow result', async () => {
    vi.mocked(runImportLocalWorkflow).mockResolvedValueOnce({
      ok: true,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      bundlePath: '/tmp/bundle.tgz',
    });

    await runImportLocal(['./legacy-skill', '--package-name', '@acme.skill/legacy-skill', '--version', '0.1.0']);

    expect(runImportLocalWorkflow).toHaveBeenCalledWith({
      path: './legacy-skill',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
    });
    expect(logSpy).toHaveBeenCalledWith('Import OK');
    expect(logSpy).toHaveBeenCalledWith('package_name: @acme.skill/legacy-skill');
    expect(logSpy).toHaveBeenCalledWith('version: 0.1.0');
  });

  it('prints failure output from the import-local workflow result', async () => {
    vi.mocked(runImportLocalWorkflow).mockResolvedValueOnce({
      ok: false,
      errors: ['packageName is required; pass --package-name when package.json is missing'],
    });

    await runImportLocal(['./legacy-skill']);

    expect(logSpy).toHaveBeenCalledWith('Import failed');
    expect(logSpy).toHaveBeenCalledWith(
      '- packageName is required; pass --package-name when package.json is missing'
    );
    expect(process.exitCode).toBe(1);
  });

  it('shows dedicated import-local help output', () => {
    const output = stripLogo(runCliOutput(['import-local', '--help']));
    expect(output).toContain('Usage: skills import-local [path] [options]');
    expect(output).toContain('Upload a local source directory to Catalog for normalization/import, then publish it.');
    expect(output).toContain('--package-name <name>');
    expect(output).toContain('--version <version>');
  });
});
