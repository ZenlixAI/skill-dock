import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import * as client from '../api/client';
import { SessionProvider } from '../session/SessionProvider';
import { VersionDetailPage } from './VersionDetailPage';

vi.mock('../api/client', () => ({
  applyVersionGovernanceChange: vi.fn(),
  createReleaseRecord: vi.fn(),
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  getVersionDetail: vi.fn(),
  listGovernanceChanges: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('VersionDetailPage', () => {
  it('consumes provider session posture without issuing a page-local session fetch', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-01-01T00:00:00Z',
      registry: {
        npmRegistry: 'https://npm.internal.example.com',
        tarballUrl: 'https://npm.internal.example.com/agent-notes-1.0.0.tgz',
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

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes/versions/1.0.0']}>
          <Routes>
            <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByText('发布登记只读')).toBeInTheDocument();
    await waitFor(() => expect(client.getSession).toHaveBeenCalledTimes(1));
  });
});
