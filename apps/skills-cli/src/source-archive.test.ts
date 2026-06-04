import { spawnSync } from 'child_process';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('child_process', () => ({
  spawnSync: vi.fn(),
}));

vi.mock('fs', () => ({
  mkdtempSync: vi.fn(() => '/tmp/archive-temp'),
  readFileSync: vi.fn(() => Buffer.from('archive')),
  rmSync: vi.fn(),
}));

import { createSourceArchiveBase64 } from './source-archive.ts';

describe('source archive', () => {
  beforeEach(() => {
    vi.mocked(spawnSync).mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('disables macOS copyfile metadata when building source archives', () => {
    vi.mocked(spawnSync).mockReturnValue({
      status: 0,
      stdout: '',
      stderr: '',
    } as ReturnType<typeof spawnSync>);

    const result = createSourceArchiveBase64('/tmp/demo-skill');

    expect(result).toBe(Buffer.from('archive').toString('base64'));
    expect(vi.mocked(spawnSync).mock.calls[0]?.[2]).toMatchObject({
      env: expect.objectContaining({
        COPYFILE_DISABLE: '1',
      }),
    });
  });
});
