import { getRegistryToken, registryClientConfig } from './config';

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

export async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  if (!registryClientConfig.baseUrl) {
    throw new Error('Registry base URL is not configured.');
  }

  const hasBody = init?.body != null;
  const response = await fetch(`${registryClientConfig.baseUrl}${path}`, {
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
