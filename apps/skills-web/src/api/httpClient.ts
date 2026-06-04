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

export function normalizeRegistryError(input: {
  status?: number;
  message: string;
  requestId?: string;
}): RegistryRequestError {
  if (input.status === 401 || input.status === 403) {
    return {
      kind: 'auth',
      message: 'Current session is not allowed to perform this action.',
      requestId: input.requestId,
    };
  }

  if (input.status === 404) {
    return {
      kind: 'not_found',
      message: 'The requested registry resource was not found.',
      requestId: input.requestId,
    };
  }

  if (input.status != null && input.status >= 500) {
    return {
      kind: 'server',
      message: input.message,
      requestId: input.requestId,
    };
  }

  if (input.status == null && isNetworkMessage(input.message)) {
    return {
      kind: 'network',
      message: 'Unable to reach the registry backend.',
      requestId: input.requestId,
    };
  }

  return {
    kind: 'unknown',
    message: input.message,
    requestId: input.requestId,
  };
}

export function normalizeThrownRegistryError(error: unknown): RegistryRequestError {
  if (error instanceof RegistryHttpError) {
    return normalizeRegistryError({
      status: error.status,
      message: error.message,
      requestId: error.requestId,
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
