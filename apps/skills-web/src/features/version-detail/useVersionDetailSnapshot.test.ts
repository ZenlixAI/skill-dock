import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as client from '../../api/client';
import { useVersionDetailSnapshot } from './useVersionDetailSnapshot';

vi.mock('../../api/client', () => ({
  getVersionDetail: vi.fn(),
  listGovernanceChanges: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe('useVersionDetailSnapshot', () => {
  it('surfaces normalized install metadata and a warning tone for deprecated versions', async () => {
    vi.mocked(client.getVersionDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.2.0',
      versionStatus: 'deprecated',
      installPolicy: 'catalog_only',
      publishedAt: '2026-01-01T00:00:00Z',
      registry: {
        npmRegistry: 'http://192.168.10.127:32081/repository/skills-npm/',
        tarballUrl: 'https://npm.internal.example.com/@platform.skill/agent-notes/-/agent-notes-1.2.0.tgz',
        integrity: 'sha512-test',
        shasum: 'test',
        publishedAt: '2026-01-01T00:00:00Z',
        distTags: ['latest'],
      },
      bundle: {
        skillManifestDigest: 'sha256:test',
        components: ['commands'],
      },
    });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    const { result } = renderHook(() =>
      useVersionDetailSnapshot({
        packageName: '@platform.skill/agent-notes',
        version: '1.2.0',
        refreshKey: 0,
      })
    );

    await waitFor(() => expect(result.current.detail.loading).toBe(false));

    expect(result.current.currentVersionStatus).toBe('deprecated');
    expect(result.current.versionTone).toBe('warn');
    expect(result.current.installRegistryHost).toBe('http://192.168.10.127:32081');
    expect(result.current.installTarballUrl).toBe(
      'http://192.168.10.127:32081/@platform.skill/agent-notes/-/agent-notes-1.2.0.tgz'
    );
  });
});
