import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { SessionProvider } from '../session/SessionProvider';
import { AuthoringFlowPage } from './AuthoringFlowPage';

vi.mock('../api/client', () => ({
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  listAuditEvents: vi.fn(),
  listGovernanceChanges: vi.fn(),
  listSkills: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('AuthoringFlowPage', () => {
  it('renders provider-backed session posture without issuing a page-local session fetch', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read', 'catalog.read'],
      allowedPackages: [],
    });
    vi.mocked(client.listSkills).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.listAuditEvents).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter>
          <AuthoringFlowPage />
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findAllByText('Catalog read-only access')).not.toHaveLength(0);
    await waitFor(() => expect(client.getSession).toHaveBeenCalledTimes(1));
  });
});
