import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import * as client from '../api/client';
import { LoginPage } from '../pages/LoginPage';
import { SessionProvider } from '../session/SessionProvider';
import { AppShell } from './AppShell';

vi.mock('../api/client', () => ({
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  logoutSession: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderAppShell(pathname: string) {
  return render(
    <SessionProvider>
      <MemoryRouter initialEntries={[pathname]}>
        <AppShell>
          <div>content</div>
        </AppShell>
      </MemoryRouter>
    </SessionProvider>
  );
}

function LocationProbe() {
  const location = useLocation();
  const forceLogoutLanding =
    typeof location.state === 'object' &&
    location.state !== null &&
    'forceLogoutLanding' in location.state &&
    location.state.forceLogoutLanding === true;
  return <div>{`${location.pathname}${location.search}|forceLogoutLanding=${String(forceLogoutLanding)}`}</div>;
}

describe('AppShell', () => {
  it('renders catalog with the frontstage detail shell classes and portal-first navigation', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'reader',
      actor: 'dev@company',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });

    const { container } = renderAppShell('/catalog');

    expect(await screen.findByRole('link', { name: '跳到主要内容' })).toHaveAttribute('href', '#main-content');
    expect(container.firstElementChild).toHaveClass('app-shell', 'app-shell--frontstage', 'app-shell--detail');
    expect(screen.getByRole('link', { name: '概览' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '目录' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '控制台' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '隐私' })).toHaveAttribute('href', '/privacy');
    expect(screen.getByRole('link', { name: '条款' })).toHaveAttribute('href', '/terms');
  });

  it('renders login with the backstage console shell classes', async () => {
    vi.mocked(client.getSession).mockResolvedValue(null);

    const { container } = renderAppShell('/login');

    await screen.findByRole('link', { name: '跳到主要内容' });
    expect(container.firstElementChild).toHaveClass('app-shell', 'app-shell--backstage', 'app-shell--console');
  });

  it('shows logout without switch-account when runtime login session data exists', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'reader',
      actor: 'dev@company',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });

    renderAppShell('/catalog');

    expect(await screen.findByText('已登录：dev@company')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '切换账号' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '退出登录' })).toBeInTheDocument();
  });

  it('redirects authenticated login visits to next when no logout marker state is present', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'reader',
      actor: 'dev@company',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });

    render(
      <MemoryRouter initialEntries={['/login?next=%2Fcatalog%3Fq%3Dpdf']}>
        <SessionProvider>
          <LocationProbe />
          <Routes>
            <Route path="/catalog" element={<div>catalog destination</div>} />
            <Route path="/login" element={<LoginPage />} />
          </Routes>
        </SessionProvider>
      </MemoryRouter>
    );

    expect(await screen.findByText('catalog destination')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '登录' })).not.toBeInTheDocument();
    expect(screen.getByText('/catalog?q=pdf|forceLogoutLanding=false')).toBeInTheDocument();
  });

  it('redirects logout to the login form with the preserved current location as next', async () => {
    vi.mocked(client.getSession)
      .mockResolvedValueOnce({
        role: 'reader',
        actor: 'dev@company',
        authMechanism: 'session',
        allowedActions: ['session.read'],
        allowedPackages: [],
      })
      .mockResolvedValue(null);
    vi.mocked(client.logoutSession).mockResolvedValue(undefined);

    render(
      <MemoryRouter initialEntries={['/catalog?q=pdf']}>
        <SessionProvider>
          <LocationProbe />
          <Routes>
            <Route
              path="/catalog"
              element={
                <AppShell>
                  <div>content</div>
                </AppShell>
              }
            />
            <Route path="/login" element={<LoginPage />} />
          </Routes>
        </SessionProvider>
      </MemoryRouter>
    );

    const logoutButton = await screen.findByRole('button', { name: '退出登录' });
    expect(screen.queryByRole('button', { name: '切换账号' })).not.toBeInTheDocument();

    fireEvent.click(logoutButton);

    expect(await screen.findByRole('button', { name: '登录' })).toBeInTheDocument();
    expect(screen.getByText('/login?next=%2Fcatalog%3Fq%3Dpdf|forceLogoutLanding=true')).toBeInTheDocument();
  });
});
