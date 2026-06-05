import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  canMutateCatalog,
  canPerformAction,
  describeSessionPosture,
  describeSessionScope,
  hasPackageScope,
} from './session';
import type { RegistryApiClient, RegistryClientConfig, RegistrySession } from './types';

function createSession(allowedActions: string[], allowedPackages: string[] = []): RegistrySession {
  return {
    role: 'publisher',
    actor: 'svc-publisher',
    authMechanism: 'configured_token',
    allowedActions,
    allowedPackages,
  };
}

function createMockApiClient(
  config: RegistryClientConfig,
  getSession: RegistryApiClient['getSession']
): RegistryApiClient {
  const notUsed = vi.fn();

  return {
    getConfig: () => config,
    getSession,
    loginSession: notUsed,
    logoutSession: notUsed,
    listSkills: notUsed,
    listCuratedSkills: notUsed,
    getSkillDetail: notUsed,
    getSkillVersions: notUsed,
    getInstallTarget: notUsed,
    getVersionDetail: notUsed,
    listGovernanceChanges: notUsed,
    listAuditEvents: notUsed,
    preparePackaging: notUsed,
    validatePackaging: notUsed,
    packSkill: notUsed,
    cleanupPackagingWorkspace: notUsed,
    createReleaseRecord: notUsed,
    applySkillGovernanceChange: notUsed,
    applyVersionGovernanceChange: notUsed,
  } as unknown as RegistryApiClient;
}

describe('session helpers', () => {
  it('treats absent session as non-mutating posture', () => {
    expect(canMutateCatalog(null)).toBe(false);
    expect(describeSessionPosture(null)).toBe('No authenticated Catalog session');
  });

  it('treats read-only actions as non-mutating posture', () => {
    const session = createSession(['catalog.read', 'audit.read', 'session.read']);

    expect(canMutateCatalog(session)).toBe(false);
    expect(hasPackageScope(session)).toBe(false);
    expect(describeSessionPosture(session)).toBe('Catalog read-only access');
    expect(describeSessionScope(session)).toBeNull();
  });

  it('treats explicit mutation actions as write access', () => {
    const session = createSession(['catalog.read', 'session.read', 'catalog.release_record.create']);

    expect(canMutateCatalog(session)).toBe(true);
    expect(describeSessionPosture(session)).toBe('Catalog write access');
  });

  it('describes package-scoped write access as limited', () => {
    const session = createSession(
      ['catalog.read', 'session.read', 'catalog.release_record.create'],
      ['@platform.skill/code-review', '@platform.skill/agent-notes']
    );

    expect(canMutateCatalog(session)).toBe(true);
    expect(hasPackageScope(session)).toBe(true);
    expect(describeSessionPosture(session)).toBe('Catalog write access limited to allowed packages');
    expect(describeSessionScope(session)).toBe('scope 2 packages');
  });

  it('describes package-scoped read-only access as limited', () => {
    const session = createSession(['catalog.read', 'session.read'], ['@platform.skill/code-review']);

    expect(canMutateCatalog(session)).toBe(false);
    expect(hasPackageScope(session)).toBe(true);
    expect(describeSessionPosture(session)).toBe('Catalog read-only access limited to allowed packages');
    expect(describeSessionScope(session)).toBe('scope @platform.skill/code-review');
  });

  it('allows an action only when the package is inside the allowed package scope', () => {
    const session = createSession(
      ['catalog.read', 'session.read', 'catalog.governance.skill.apply'],
      ['@platform.skill/code-review']
    );

    expect(canPerformAction(session, 'catalog.governance.skill.apply', '@platform.skill/code-review')).toBe(true);
    expect(canPerformAction(session, 'catalog.governance.skill.apply', '@platform.skill/agent-notes')).toBe(false);
    expect(canPerformAction(session, 'catalog.governance.skill.apply')).toBe(false);
  });

  it('honors wildcard package scope patterns', () => {
    const session = createSession(
      ['catalog.read', 'session.read', 'catalog.governance.version.apply'],
      ['@platform.skill/*']
    );

    expect(canPerformAction(session, 'catalog.governance.version.apply', '@platform.skill/code-review')).toBe(true);
    expect(canPerformAction(session, 'catalog.governance.version.apply', '@partner.skill/code-review')).toBe(false);
  });
});

describe('session client seam', () => {
  afterEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    vi.unmock('./client');
    vi.unmock('./config');
    vi.unmock('./httpClient');
    vi.unmock('./mockClient');
  });

  it('returns null in mock mode through the main api client seam', async () => {
    const mockGetSession = vi.fn().mockResolvedValue(null);
    const mockConfig: RegistryClientConfig = {
      mode: 'mock',
      baseUrl: null,
      role: null,
      actor: null,
      hasToken: false,
    };

    vi.doMock('./config', () => ({
      registryClientConfig: mockConfig,
    }));
    vi.doMock('./mockClient', () => ({
      mockClient: createMockApiClient(mockConfig, mockGetSession),
    }));
    vi.doMock('./httpClient', () => ({
      requestJSON: vi.fn(),
    }));

    const clientModule = await import('./client');

    await expect(clientModule.getSession()).resolves.toBeNull();
    expect(mockGetSession).toHaveBeenCalledTimes(1);
  });

  it('uses the real client seam to request the current session in real mode', async () => {
    const requestJSON = vi.fn().mockResolvedValue({
      role: 'authoring',
      actor: 'svc-authoring',
      authMechanism: 'configured_token',
      allowedActions: ['session.read', 'packaging.prepare'],
      allowedPackages: ['@platform.skill/code-review'],
    });
    const mockConfig: RegistryClientConfig = {
      mode: 'real',
      baseUrl: 'http://127.0.0.1:18080',
      role: null,
      actor: null,
      hasToken: true,
    };

    vi.doMock('./config', () => ({
      registryClientConfig: mockConfig,
    }));
    vi.doMock('./httpClient', () => ({
      requestJSON,
    }));
    vi.doMock('./mockClient', () => ({
      mockClient: createMockApiClient(
        {
          mode: 'mock',
          baseUrl: null,
          role: null,
          actor: null,
          hasToken: false,
        },
        vi.fn()
      ),
    }));

    const clientModule = await import('./client');

    await expect(clientModule.getSession()).resolves.toEqual({
      role: 'authoring',
      actor: 'svc-authoring',
      authMechanism: 'configured_token',
      allowedActions: ['session.read', 'packaging.prepare'],
      allowedPackages: ['@platform.skill/code-review'],
    });
    expect(requestJSON).toHaveBeenCalledWith('/api/v1/session');
  });

  it('uses the real client seam even when baseUrl is omitted for same-origin runtime', async () => {
    const requestJSON = vi.fn().mockResolvedValue({
      role: 'authoring',
      actor: 'svc-authoring',
      authMechanism: 'configured_token',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    const mockConfig: RegistryClientConfig = {
      mode: 'real',
      baseUrl: null,
      role: null,
      actor: null,
      hasToken: true,
    };

    vi.doMock('./config', () => ({
      registryClientConfig: mockConfig,
    }));
    vi.doMock('./httpClient', () => ({
      requestJSON,
    }));
    vi.doMock('./mockClient', () => ({
      mockClient: createMockApiClient(
        {
          mode: 'mock',
          baseUrl: null,
          role: null,
          actor: null,
          hasToken: false,
        },
        vi.fn()
      ),
    }));

    const clientModule = await import('./client');

    await expect(clientModule.getSession()).resolves.toEqual({
      role: 'authoring',
      actor: 'svc-authoring',
      authMechanism: 'configured_token',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    expect(requestJSON).toHaveBeenCalledWith('/api/v1/session');
  });

  it('treats an unauthenticated real-mode session response as null', async () => {
    const requestJSON = vi.fn().mockResolvedValue(null);
    const mockConfig: RegistryClientConfig = {
      mode: 'real',
      baseUrl: null,
      role: null,
      actor: null,
      hasToken: false,
    };

    vi.doMock('./config', () => ({
      registryClientConfig: mockConfig,
    }));
    vi.doMock('./httpClient', () => ({
      requestJSON,
      RegistryHttpError: class RegistryHttpError extends Error {},
    }));
    vi.doMock('./mockClient', () => ({
      mockClient: createMockApiClient(
        {
          mode: 'mock',
          baseUrl: null,
          role: null,
          actor: null,
          hasToken: false,
        },
        vi.fn()
      ),
    }));

    const clientModule = await import('./client');

    await expect(clientModule.getSession()).resolves.toBeNull();
    expect(requestJSON).toHaveBeenCalledWith('/api/v1/session');
  });
});
