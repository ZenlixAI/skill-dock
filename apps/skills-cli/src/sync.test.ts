import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@clack/prompts', () => ({
  select: vi.fn(),
  isCancel: vi.fn(() => false),
  cancel: vi.fn(),
}));

vi.mock('./sync-workflow.ts', () => ({
  runSyncWorkflow: vi.fn(),
}));
vi.mock('./update.ts', () => ({
  hasProjectSkills: vi.fn(() => true),
}));

import { parseSyncOptions, runSync } from './sync.ts';
import { runSyncWorkflow } from './sync-workflow.ts';
import { runCliOutput, stripLogo } from './test-utils.ts';
import { hasProjectSkills } from './update.ts';

describe('sync command', () => {
  let logSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.mocked(runSyncWorkflow).mockReset();
    process.exitCode = undefined;
  });

  afterEach(() => {
    logSpy.mockRestore();
    vi.restoreAllMocks();
    process.exitCode = undefined;
    Object.defineProperty(process.stdin, 'isTTY', {
      value: true,
      configurable: true,
    });
  });

  it('parses explicit sync options', () => {
    expect(parseSyncOptions(['foo', 'bar', '--global', '--yes'])).toEqual({
      skills: ['foo', 'bar'],
      global: true,
      yes: true,
    });
  });

  it('prints success output from the sync workflow result', async () => {
    vi.mocked(runSyncWorkflow).mockResolvedValueOnce({
      successCount: 1,
      failCount: 0,
      skippedCount: 1,
      failures: [],
      skipped: ['@acme.skill/registry-only'],
    });

    await runSync(['legacy-skill', '--project']);

    expect(runSyncWorkflow).toHaveBeenCalledWith({
      project: true,
      skills: ['legacy-skill'],
    });
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Checking source-installed skills'));
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('✓ Synced 1 skill(s)'));
  });

  it('defaults to global scope when no project skills are installed in non-interactive mode', async () => {
    vi.mocked(hasProjectSkills).mockReturnValueOnce(false);
    Object.defineProperty(process.stdin, 'isTTY', {
      value: false,
      configurable: true,
    });
    vi.mocked(runSyncWorkflow).mockResolvedValueOnce({
      successCount: 0,
      failCount: 0,
      skippedCount: 0,
      failures: [],
      skipped: [],
    });

    await runSync([]);

    expect(runSyncWorkflow).toHaveBeenCalledWith({
      global: true,
    });
  });

  it('prints failure output when sync workflow reports failures', async () => {
    vi.mocked(runSyncWorkflow).mockResolvedValueOnce({
      successCount: 0,
      failCount: 1,
      skippedCount: 0,
      failures: [{ skill: 'legacy-skill', error: 'legacy.source is required for sync' }],
      skipped: [],
    });

    await runSync(['legacy-skill']);

    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Failed to sync 1 skill(s)'));
    expect(process.exitCode).toBe(1);
  });

  it('shows dedicated sync help output', () => {
    const output = stripLogo(runCliOutput(['sync', '--help']));
    expect(output).toContain('Usage: skills sync [skills...] [options]');
    expect(output).toContain('Refresh installed skills that came from local paths or git sources.');
    expect(output).toContain('--global');
    expect(output).toContain('--project');
  });
});
