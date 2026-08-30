import { beforeEach, describe, expect, it, vi } from 'vitest';
import { rmSync } from 'fs';

const cloneMock = vi.fn();

vi.mock('simple-git', () => ({
  default: vi.fn(() => ({
    env: vi.fn(() => ({
      clone: cloneMock,
    })),
  })),
}));

import { cloneRepo } from './git.ts';

describe('cloneRepo', () => {
  beforeEach(() => {
    cloneMock.mockReset();
  });

  it('retries without shallow cloning when dumb HTTP rejects depth 1', async () => {
    cloneMock
      .mockRejectedValueOnce(new Error('fatal: dumb http transport does not support shallow capabilities'))
      .mockResolvedValueOnce(undefined);

    const tempDir = await cloneRepo('http://example.com/repo.git');

    expect(cloneMock).toHaveBeenNthCalledWith(1, 'http://example.com/repo.git', expect.any(String), [
      '--depth',
      '1',
    ]);
    expect(cloneMock).toHaveBeenNthCalledWith(2, 'http://example.com/repo.git', expect.any(String), []);

    rmSync(tempDir, { recursive: true, force: true });
  });
});
