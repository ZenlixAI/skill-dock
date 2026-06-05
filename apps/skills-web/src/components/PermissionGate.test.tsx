import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as client from '../api/client';
import type { RegistrySession } from '../api/types';
import { SessionProvider } from '../session/SessionProvider';
import { PermissionGate } from './PermissionGate';

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

function createSession(allowedActions: string[], allowedPackages: string[] = []): RegistrySession {
  return {
    role: 'publisher',
    actor: 'svc-publisher',
    authMechanism: 'configured_token',
    allowedActions,
    allowedPackages,
  };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('PermissionGate', () => {
  it('renders the fallback posture when the session lacks the required action', () => {
    render(
      <PermissionGate
        fallback={<div>当前会话对 packaging.prepare 只有只读权限。</div>}
        requiredAction="packaging.prepare"
        session={createSession(['catalog.read', 'session.read'])}
      >
        <button type="button">Prepare packaging</button>
      </PermissionGate>
    );

    expect(screen.getByText('当前会话对 packaging.prepare 只有只读权限。')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Prepare packaging' })).toBeNull();
  });

  it('renders children when the session includes the required action', () => {
    render(
      <PermissionGate requiredAction="packaging.prepare" session={createSession(['session.read', 'packaging.prepare'])}>
        <button type="button">Prepare packaging</button>
      </PermissionGate>
    );

    expect(screen.getByRole('button', { name: 'Prepare packaging' })).toBeInTheDocument();
  });

  it('blocks package-scoped sessions when the package is outside the allowed scope', () => {
    render(
      <PermissionGate
        packageName="@platform.skill/agent-notes"
        requiredAction="catalog.governance.skill.apply"
        session={createSession(
          ['session.read', 'catalog.governance.skill.apply'],
          ['@platform.skill/code-review']
        )}
      >
        <button type="button">Retire skill</button>
      </PermissionGate>
    );

    expect(screen.getByText('当前会话没有执行 catalog.governance.skill.apply（@platform.skill/agent-notes）的权限。')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Retire skill' })).toBeNull();
  });

  it('renders an unavailable posture when the session check failed', () => {
    render(
      <PermissionGate requiredAction="packaging.prepare" session={null} sessionError="registry offline">
        <button type="button">Prepare packaging</button>
      </PermissionGate>
    );

    expect(screen.getByText('当前会话不可用，暂时无法校验 packaging.prepare。')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Prepare packaging' })).toBeNull();
  });

  it('defaults to provider loading state when explicit session props are omitted', () => {
    vi.mocked(client.getSession).mockImplementation(() => new Promise(() => undefined));

    render(
      <SessionProvider>
        <PermissionGate requiredAction="packaging.prepare">
          <button type="button">Prepare packaging</button>
        </PermissionGate>
      </SessionProvider>
    );

    expect(screen.getByText('正在加载会话权限…')).toBeInTheDocument();
    expect(screen.queryByText('当前没有可用于 packaging.prepare 的已登录 Catalog 会话。')).toBeNull();
  });

  it('resolves missing session state props from the provider when overrides are partial', () => {
    vi.mocked(client.getSession).mockImplementation(() => new Promise(() => undefined));

    render(
      <SessionProvider>
        <PermissionGate
          requiredAction="packaging.prepare"
          session={createSession(['session.read', 'packaging.prepare'])}
        >
          <button type="button">Prepare packaging</button>
        </PermissionGate>
      </SessionProvider>
    );

    expect(screen.getByText('正在加载会话权限…')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Prepare packaging' })).toBeNull();
  });

  it('lets an explicit null session override win over provider state', async () => {
    vi.mocked(client.getSession).mockResolvedValue(
      createSession(['session.read', 'packaging.prepare'])
    );

    render(
      <SessionProvider>
        <PermissionGate requiredAction="packaging.prepare" session={null}>
          <button type="button">Prepare packaging</button>
        </PermissionGate>
      </SessionProvider>
    );

    expect(
      await screen.findByText('当前没有可用于 packaging.prepare 的已登录 Catalog 会话。')
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Prepare packaging' })).toBeNull();
  });
});
