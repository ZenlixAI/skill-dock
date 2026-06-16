import { getRegistryToken, registryClientConfig } from './config';
import type { RegistryRequestError } from './types';

interface CatalogErrorPayload {
  error?: {
    code?: string;
    message?: string;
    details?: Record<string, unknown>;
    requestId?: string;
  };
}

export class RegistryHttpError extends Error {
  status: number;
  code?: string;
  details?: Record<string, unknown>;
  requestId?: string;

  constructor(message: string, status: number, payload?: CatalogErrorPayload) {
    super(message);
    this.name = 'RegistryHttpError';
    this.status = status;
    this.code = payload?.error?.code;
    this.details = payload?.error?.details;
    this.requestId = payload?.error?.requestId;
  }
}

function buildHeaders(hasBody: boolean) {
  const headers = new Headers();
  headers.set('Accept', 'application/json');
  if (hasBody) {
    headers.set('Content-Type', 'application/json');
  }

  const token = getRegistryToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  } else if (registryClientConfig.role) {
    headers.set('X-Role', registryClientConfig.role);
  }

  if (registryClientConfig.actor) {
    headers.set('X-Actor', registryClientConfig.actor);
  }

  return headers;
}

function toErrorMessage(status: number, payload?: CatalogErrorPayload) {
  const code = payload?.error?.code;
  const message = payload?.error?.message;
  if (code && message) {
    return `${code}: ${message}`;
  }
  if (message) {
    return message;
  }
  return `Request failed with status ${status}.`;
}

function isNetworkMessage(message: string) {
  const normalized = message.toLowerCase();
  return (
    normalized.includes('failed to fetch') ||
    normalized.includes('networkerror') ||
    normalized.includes('network error') ||
    normalized.includes('load failed') ||
    normalized.includes('fetch failed')
  );
}

function isMissingPackagingWorkspaceMessage(message: string) {
  return message.toLowerCase().includes('workspacepath must reference an existing directory');
}

function isRegistryPublishAuthFailure(code: string | undefined, details: Record<string, unknown> | undefined) {
  if (code !== 'registry_publish_failed' || !details) {
    return false;
  }
  const status = details.status;
  return status === 401 || status === 403;
}

export function normalizeRegistryError(input: {
  status?: number;
  message: string;
  code?: string;
  requestId?: string;
  details?: Record<string, unknown>;
}): RegistryRequestError {
  if (input.status === 401 || input.status === 403) {
    return {
      kind: 'auth',
      message: 'Current session is not allowed to perform this action.',
      requestId: input.requestId,
      code: input.code,
      status: input.status,
      details: input.details,
    };
  }

  if (input.status === 404) {
    if (input.code === 'registry_version_not_found') {
      return {
        kind: 'not_found',
        message: 'Registry does not contain this package version yet. Publish the packed bundle to the npm registry, then retry release-record.',
        requestId: input.requestId,
        code: input.code,
        status: input.status,
        details: input.details,
      };
    }
    return {
      kind: 'not_found',
      message: 'The requested registry resource was not found.',
      requestId: input.requestId,
      code: input.code,
      status: input.status,
      details: input.details,
    };
  }

  if (input.status != null && input.status >= 500) {
    if (isRegistryPublishAuthFailure(input.code, input.details)) {
      return {
        kind: 'server',
        message:
          'Catalog tried to publish to the npm registry, but the registry rejected the backend credentials. Check `catalogCommon.auth.npmPublishAuthorization` / `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`, then retry.',
        requestId: input.requestId,
        code: input.code,
        status: input.status,
        details: input.details,
      };
    }
    return {
      kind: 'server',
      message: input.message,
      requestId: input.requestId,
      code: input.code,
      status: input.status,
      details: input.details,
    };
  }

  if (input.status === 422 && isMissingPackagingWorkspaceMessage(input.message)) {
    return {
      kind: 'unknown',
      message:
        'Catalog returned a legacy workspace-path error. One-shot packaging should be stateless; retry after upgrading Catalog to the latest packaging API.',
      requestId: input.requestId,
      code: input.code,
      status: input.status,
      details: input.details,
    };
  }

  if (input.status == null && isNetworkMessage(input.message)) {
    return {
      kind: 'network',
      message: 'Unable to reach the registry backend.',
      requestId: input.requestId,
      code: input.code,
      status: input.status,
      details: input.details,
    };
  }

  return {
    kind: 'unknown',
    message: input.message,
    requestId: input.requestId,
    code: input.code,
    status: input.status,
    details: input.details,
  };
}

export function normalizeThrownRegistryError(error: unknown): RegistryRequestError {
  if (error instanceof RegistryHttpError) {
    return normalizeRegistryError({
      status: error.status,
      message: error.message,
      code: error.code,
      requestId: error.requestId,
      details: error.details,
    });
  }

  if (error instanceof Error) {
    return normalizeRegistryError({
      message: error.message,
    });
  }

  return normalizeRegistryError({
    message: String(error),
  });
}

export async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const hasBody = init?.body != null;
  const requestUrl = registryClientConfig.baseUrl ? `${registryClientConfig.baseUrl}${path}` : path;
  const response = await fetch(requestUrl, {
    ...init,
    credentials: 'include',
    headers: buildHeaders(hasBody),
  });

  const isJSON = response.headers.get('content-type')?.includes('application/json');
  const payload = isJSON ? ((await response.json()) as T | CatalogErrorPayload) : null;

  if (!response.ok) {
    throw new RegistryHttpError(
      toErrorMessage(response.status, (payload as CatalogErrorPayload) || undefined),
      response.status,
      (payload as CatalogErrorPayload) || undefined
    );
  }

  return payload as T;
}
