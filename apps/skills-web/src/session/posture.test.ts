import { describe, expect, it } from 'vitest';
import type { RegistrySession } from '../api/types';
import {
  getSessionPosture,
  hasCapability,
  isSessionUnavailable,
} from './posture';

function createSession(actions: string[], allowedPackages: string[] = []): RegistrySession {
  return {
    role: 'operator',
    actor: 'tester',
    authMechanism: 'header',
    allowedActions: actions,
    allowedPackages,
  };
}

describe('session posture', () => {
  it('returns package_filter_required when a scoped governance session has no package focus', () => {
    const session = createSession(['catalog.governance.skill.apply'], ['@platform.skill/*']);

    expect(
      getSessionPosture(session, {
        packageName: '',
        requiredAction: 'catalog.governance.skill.apply',
      })
    ).toBe('package_filter_required');
  });

  it('allows a scoped package write when the package matches', () => {
    const session = createSession(['catalog.governance.skill.apply'], ['@platform.skill/*']);

    expect(
      hasCapability(session, 'catalog.governance.skill.apply', '@platform.skill/pdf-skill')
    ).toBe(true);
  });

  it('does not require a package filter when the session lacks the required action', () => {
    const session = createSession(['session.read'], ['@platform.skill/*']);

    expect(
      getSessionPosture(session, {
        packageName: '',
        requiredAction: 'catalog.governance.skill.apply',
      })
    ).toBe('read_only');
  });

  it('marks an errored session load as unavailable', () => {
    expect(isSessionUnavailable(false, 'network down')).toBe(true);
    expect(isSessionUnavailable(true, 'network down')).toBe(false);
  });
});
