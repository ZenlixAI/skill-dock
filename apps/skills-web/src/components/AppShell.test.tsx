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

    expect(await screen.findByRole('link', { name: '跳到主要内容' })).toHaveAttribute('href', '#main-content');
    expect(screen.getByRole('link', { name: '概览' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '目录' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '控制台' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '隐私' })).toHaveAttribute('href', '/privacy');
    expect(screen.getByRole('link', { name: '条款' })).toHaveAttribute('href', '/terms');
  });
});
