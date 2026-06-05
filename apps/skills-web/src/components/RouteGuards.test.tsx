import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import * as client from '../api/client';
import { RegistryHttpError } from '../api/httpClient';
import { SessionProvider } from '../session/SessionProvider';
import { RequireAuthenticatedSession } from './RouteGuards';

vi.mock('../api/client', () => ({
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
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
  it('redirects to login when the provider session check fails with an auth error', async () => {
    vi.mocked(client.getSession).mockRejectedValue(
      new RegistryHttpError('Unauthorized', 401, {
        error: {
          code: 'unauthorized',
          message: 'Unauthorized',
        },
      })
    );

    render(
      <MemoryRouter initialEntries={['/protected?tab=review']}>
        <SessionProvider>
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
        </SessionProvider>
      </MemoryRouter>
    );

    expect(await screen.findByText('login?next=%2Fprotected%3Ftab%3Dreview')).toBeInTheDocument();
    expect(screen.queryByText('secret')).toBeNull();
  });
});
