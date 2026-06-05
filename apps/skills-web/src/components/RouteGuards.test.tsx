import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { getRegistryClientConfig } from '../api/client';
import { useRegistrySession } from '../session/SessionProvider';
import { RequireAuthenticatedSession } from './RouteGuards';

vi.mock('../api/client', () => ({
  getRegistryClientConfig: vi.fn(),
}));

vi.mock('../session/SessionProvider', () => ({
  useRegistrySession: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function LoginMarker() {
  const location = useLocation();
  return <div>{`login${location.search}`}</div>;
}

describe('RequireAuthenticatedSession', () => {
  it('redirects to login when the session check fails with an auth error', () => {
    vi.mocked(getRegistryClientConfig).mockReturnValue({
      mode: 'real',
      baseUrl: null,
      role: null,
      actor: null,
      hasToken: false,
    });
    vi.mocked(useRegistrySession).mockReturnValue({
      session: {
        data: null,
        loading: false,
        refreshing: false,
        error: 'Unauthorized',
        errorDetail: {
          kind: 'auth',
          message: 'Unauthorized',
        },
      },
      refreshSession: vi.fn(),
      getPosture: () => 'unavailable',
      getPostureLabel: () => '会话不可用',
      hasCapability: () => false,
    });

    render(
      <MemoryRouter initialEntries={['/protected?tab=review']}>
        <Routes>
          <Route
            path="/protected"
            element={
              <RequireAuthenticatedSession>
                <div>secret</div>
              </RequireAuthenticatedSession>
            }
          />
          <Route path="/login" element={<LoginMarker />} />
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('login?next=%2Fprotected%3Ftab%3Dreview')).toBeInTheDocument();
    expect(screen.queryByText('secret')).toBeNull();
  });
});
