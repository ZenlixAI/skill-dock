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
        fallback={<div>Current session is read-only for packaging.prepare.</div>}
        requiredAction="packaging.prepare"
        session={createSession(['catalog.read', 'session.read'])}
      >
        <button type="button">Prepare packaging</button>
      </PermissionGate>
    );

    expect(screen.getByText('Current session is read-only for packaging.prepare.')).toBeInTheDocument();
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

    expect(screen.getByText('Current session cannot perform catalog.governance.skill.apply for @platform.skill/agent-notes.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Retire skill' })).toBeNull();
  });

  it('renders an unavailable posture when the session check failed', () => {
    render(
      <PermissionGate requiredAction="packaging.prepare" session={null} sessionError="registry offline">
        <button type="button">Prepare packaging</button>
      </PermissionGate>
    );

    expect(screen.getByText('Session unavailable. Unable to verify packaging.prepare right now.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Prepare packaging' })).toBeNull();
  });
});
