import { afterEach, describe, expect, it, vi } from 'vitest';
import { RegistryHttpError, normalizeRegistryError, normalizeThrownRegistryError } from './httpClient';

afterEach(() => {
  vi.resetModules();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe('normalizeRegistryError', () => {
  it('turns auth statuses into auth-visible errors', () => {
    expect(normalizeRegistryError({ status: 401, message: 'unauthorized' }).kind).toBe('auth');
    expect(normalizeRegistryError({ status: 403, message: 'forbidden' }).kind).toBe('auth');
  });

  it('turns 404 into a not_found error', () => {
    expect(normalizeRegistryError({ status: 404, message: 'missing' })).toEqual({
      kind: 'not_found',
      message: 'The requested registry resource was not found.',
      requestId: undefined,
      code: undefined,
      status: 404,
      details: undefined,
    });
  });

  it('explains when release-record fails because the registry version was never published', () => {
    expect(
      normalizeRegistryError({
        status: 404,
        code: 'registry_version_not_found',
        message: 'registry_version_not_found: Registry version not found.',
      })
    ).toEqual({
      kind: 'not_found',
      message: 'Registry does not contain this package version yet. Publish the packed bundle to the npm registry, then retry release-record.',
      requestId: undefined,
      code: 'registry_version_not_found',
      status: 404,
      details: undefined,
    });
  });

  it('turns 5xx responses into server errors and keeps the request id', () => {
    expect(normalizeRegistryError({ status: 503, message: 'upstream unavailable', requestId: 'req-123' })).toEqual({
      kind: 'server',
      message: 'upstream unavailable',
      requestId: 'req-123',
      code: undefined,
      status: 503,
      details: undefined,
    });
  });

  it('turns registry_publish_failed with upstream 401 details into an actionable publish-auth hint', () => {
    expect(
      normalizeRegistryError({
        status: 503,
        code: 'registry_publish_failed',
        message: 'registry_publish_failed: Registry publish request failed.',
        requestId: 'req-1264',
        details: {
          status: 401,
          responseBody: 'Unauthorized',
        },
      })
    ).toEqual({
      kind: 'server',
      message:
        'Catalog tried to publish to the npm registry, but the registry rejected the backend credentials. Check `catalogCommon.auth.npmPublishAuthorization` / `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`, then retry.',
      requestId: 'req-1264',
      code: 'registry_publish_failed',
      status: 503,
      details: {
        status: 401,
        responseBody: 'Unauthorized',
      },
    });
  });

  it('turns fetch failures into network errors', () => {
    expect(normalizeRegistryError({ message: 'Failed to fetch' })).toEqual({
      kind: 'network',
      message: 'Unable to reach the registry backend.',
      requestId: undefined,
      code: undefined,
      status: undefined,
      details: undefined,
    });
  });

  it('falls back to unknown when there is no stronger signal', () => {
    expect(normalizeRegistryError({ message: 'something odd happened' })).toEqual({
      kind: 'unknown',
      message: 'something odd happened',
      requestId: undefined,
      code: undefined,
      status: undefined,
      details: undefined,
    });
  });

  it('turns missing packaging workspace errors into an actionable operator hint', () => {
    expect(
      normalizeRegistryError({
        status: 422,
        message: 'invalid_request: workspacePath must reference an existing directory.',
        requestId: 'req-789',
      })
    ).toEqual({
      kind: 'unknown',
      message:
        'Packaging workspace is no longer available. If Catalog is running with multiple replicas, scale `catalog` down to 1 replica for prepare/validate/pack flows, then retry.',
      requestId: 'req-789',
      code: undefined,
      status: 422,
      details: undefined,
    });
  });
});

describe('normalizeThrownRegistryError', () => {
  it('converts RegistryHttpError instances through the normalizer', () => {
    const httpError = new RegistryHttpError('service unavailable', 503, {
      error: {
        requestId: 'req-456',
      },
    });

    expect(normalizeThrownRegistryError(httpError)).toEqual({
      kind: 'server',
      message: 'service unavailable',
      requestId: 'req-456',
      code: undefined,
      status: 503,
      details: undefined,
    });
  });
});

describe('requestJSON', () => {
  it('uses same-origin paths when baseUrl is omitted in real mode', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({ ok: true }),
    });
    vi.stubGlobal('fetch', fetchMock);

    vi.doMock('./config', () => ({
      registryClientConfig: {
        mode: 'real',
        baseUrl: null,
        role: null,
        actor: null,
        hasToken: false,
      },
      getRegistryToken: () => null,
    }));

    const { requestJSON } = await import('./httpClient');

    await expect(requestJSON('/api/v1/session')).resolves.toEqual({ ok: true });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/session',
      expect.objectContaining({
        headers: expect.any(Headers),
      })
    );
  });
});
