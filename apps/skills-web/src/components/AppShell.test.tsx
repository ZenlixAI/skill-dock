import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
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

describe('AppShell', () => {
  it('renders a skip link, portal-first nav, console entry, and legal footer links', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'reader',
      actor: 'dev@company',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/catalog']}>
          <AppShell>
            <div>content</div>
          </AppShell>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#main-content');
    expect(screen.getByRole('link', { name: 'Overview' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Catalog' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Console' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Privacy' })).toHaveAttribute('href', '/privacy');
    expect(screen.getByRole('link', { name: 'Terms' })).toHaveAttribute('href', '/terms');
  });
});
