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
    });
  });

  it('turns 5xx responses into server errors and keeps the request id', () => {
    expect(normalizeRegistryError({ status: 503, message: 'upstream unavailable', requestId: 'req-123' })).toEqual({
      kind: 'server',
      message: 'upstream unavailable',
      requestId: 'req-123',
    });
  });

  it('turns fetch failures into network errors', () => {
    expect(normalizeRegistryError({ message: 'Failed to fetch' })).toEqual({
      kind: 'network',
      message: 'Unable to reach the registry backend.',
      requestId: undefined,
    });
  });

  it('falls back to unknown when there is no stronger signal', () => {
    expect(normalizeRegistryError({ message: 'something odd happened' })).toEqual({
      kind: 'unknown',
      message: 'something odd happened',
      requestId: undefined,
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
