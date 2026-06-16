import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as client from '../../api/client';
import { useSkillDetailSnapshot } from './useSkillDetailSnapshot';

vi.mock('../../api/client', () => ({
  getInstallTarget: vi.fn(),
  getSkillDetail: vi.fn(),
  getSkillVersions: vi.fn(),
  listGovernanceChanges: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe('useSkillDetailSnapshot', () => {
  it('surfaces the install target version item instead of assuming the latest version is installable', async () => {
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: '2.0.0',
      latestVersionStatus: 'published',
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockResolvedValue({
      items: [
        {
          packageName: '@platform.skill/agent-notes',
          version: '2.0.0',
          versionStatus: 'published',
          installPolicy: 'blocked',
          publishedAt: '2026-02-01T00:00:00Z',
        },
        {
          packageName: '@platform.skill/agent-notes',
          version: '1.9.0',
          versionStatus: 'published',
          installPolicy: 'resolvable',
          publishedAt: '2026-01-15T00:00:00Z',
        },
      ],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.getInstallTarget).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.9.0',
      npmRegistry: 'http://registry.internal',
      tarballUrl: 'https://registry.internal/@platform.skill/agent-notes/-/agent-notes-1.9.0.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256:test',
      versionStatus: 'published',
    });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    const { result } = renderHook(() =>
      useSkillDetailSnapshot({
        packageName: '@platform.skill/agent-notes',
        refreshKey: 0,
      })
    );

    await waitFor(() => expect(result.current.installTarget.loading).toBe(false));

    expect(result.current.latestVersionItem?.version).toBe('2.0.0');
    expect(result.current.installTargetVersionItem?.version).toBe('1.9.0');
    expect(result.current.installTargetVersionItem?.installPolicy).toBe('resolvable');
    expect(result.current.installVersion).toBe('1.9.0');
    expect(result.current.hasInstallTarget).toBe(true);
    expect(result.current.statusTone).toBe('good');
  });
});
