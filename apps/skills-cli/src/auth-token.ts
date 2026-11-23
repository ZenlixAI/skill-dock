import { readCredentials, writeCredentials, type StoredCredentials } from './credentials.ts';

export async function getStoredAccessToken(): Promise<string | undefined> {
  const credentials = readCredentials();
  if (!credentials) return undefined;
  if (credentials.accessToken && credentials.accessTokenExpiresAt && credentials.accessTokenExpiresAt > Date.now() + 60_000) {
    return credentials.accessToken;
  }
  return refreshCredentials(credentials);
}

export async function refreshCredentials(credentials: StoredCredentials): Promise<string | undefined> {
  try {
    const response = await fetch(`${credentials.logtoEndpoint}/oidc/token`, {
      method: 'POST',
      headers: { 'content-type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        client_id: credentials.clientId,
        grant_type: 'refresh_token',
        refresh_token: credentials.refreshToken,
        ...(credentials.apiResource ? { resource: credentials.apiResource } : {}),
        ...(credentials.organizationId ? { organization_id: credentials.organizationId } : {}),
      }).toString(),
    });
    if (!response.ok) return undefined;
    const result = (await response.json()) as Record<string, unknown>;
    if (typeof result.error === 'string') return undefined;
    const next: StoredCredentials = {
      ...credentials,
      accessToken: String(result.access_token || ''),
      accessTokenExpiresAt: Date.now() + Number(result.expires_in || 3600) * 1000,
      refreshToken: String(result.refresh_token || credentials.refreshToken),
    };
    writeCredentials(next);
    return next.accessToken;
  } catch {
    return undefined;
  }
}
