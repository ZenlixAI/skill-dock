import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import type { RegistrySession } from '../api/types';
import { PermissionGate } from './PermissionGate';

function createSession(allowedActions: string[]): RegistrySession {
  return {
    role: 'publisher',
    actor: 'svc-publisher',
    authMechanism: 'configured_token',
    allowedActions,
    allowedPackages: [],
  };
}

afterEach(() => {
  cleanup();
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
        session={{
          ...createSession(['session.read', 'catalog.governance.skill.apply']),
          allowedPackages: ['@platform.skill/code-review'],
        }}
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
});
