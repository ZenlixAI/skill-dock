import { getStoredAccessToken } from './auth-token.ts';
import { clearCredentials, cliClientId, logtoEndpoint, readCredentials, writeCredentials, type StoredCredentials } from './credentials.ts';
import { getRegistryBaseUrl } from './catalog.ts';

const DEVICE_SCOPE = [
  'openid',
  'profile',
  'email',
  'offline_access',
  'urn:logto:scope:organizations',
  'catalog:read',
  'catalog:install',
  'skill:publish',
  'favorites:read',
  'favorites:write',
].join(' ');

export interface LoginOptions {
  logtoEndpoint?: string;
  clientId?: string;
  apiResource?: string;
}

export async function runLogin(options: LoginOptions = {}): Promise<void> {
  const endpoint = (options.logtoEndpoint || logtoEndpoint()).replace(/\/+$/, '');
  const clientId = options.clientId || cliClientId();
  const resource = options.apiResource || (process.env.SKILLS_LOGTO_API_RESOURCE || '').trim();
  if (!endpoint || !clientId || !resource) {
    throw new Error(
      'Logto configuration is required. Set SKILLS_LOGTO_ENDPOINT, SKILLS_LOGTO_CLIENT_ID, and SKILLS_LOGTO_API_RESOURCE, or pass --logto-endpoint, --client-id, and --api-resource.'
    );
  }

  const device = await requestJSON(`${endpoint}/oidc/device/auth`, {
    method: 'POST',
    headers: { 'content-type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      client_id: clientId,
      scope: DEVICE_SCOPE,
      ...(resource ? { resource } : {}),
    }).toString(),
  });
  const userCode = String(device.user_code || '');
  const verificationURI = String(device.verification_uri_complete || device.verification_uri || '');
  const deviceCode = String(device.device_code || '');
  const expiresIn = Number(device.expires_in || 300);
  if (!deviceCode || !userCode || !verificationURI) {
    throw new Error('Logto did not return a usable device authorization response.');
  }

  console.log(`Open ${verificationURI}`);
  console.log(`Enter code: ${userCode}`);

  const deadline = Date.now() + expiresIn * 1000;
  let intervalSeconds = Number(device.interval || 5);
  while (Date.now() < deadline) {
    await sleep(intervalSeconds * 1000);
    const result = await requestJSON(`${endpoint}/oidc/token`, {
      method: 'POST',
      headers: { 'content-type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        client_id: clientId,
        grant_type: 'urn:ietf:params:oauth:grant-type:device_code',
        device_code: deviceCode,
      }).toString(),
    });
    if (result.error === 'authorization_pending') continue;
    if (result.error === 'slow_down') {
      intervalSeconds += 5;
      continue;
    }
    if (result.error) {
      throw new Error(`Logto device flow failed: ${result.error_description || result.error}`);
    }
    const credentials: StoredCredentials = {
      logtoEndpoint: endpoint,
      clientId,
      apiResource: resource || undefined,
      refreshToken: String(result.refresh_token || ''),
      accessToken: String(result.access_token || ''),
      accessTokenExpiresAt: Date.now() + Number(result.expires_in || 3600) * 1000,
    };
    if (!credentials.refreshToken) {
      throw new Error('Logto did not return a refresh token. Ensure offline_access is enabled.');
    }
    writeCredentials(credentials);
    console.log('Signed in.');
    return;
  }
  throw new Error('Logto device code expired before sign-in completed.');
}

export async function runWhoami(options: { registry?: string } = {}): Promise<void> {
  const me = await fetchMe(options.registry);
  console.log(JSON.stringify(me, null, 2));
}

export async function runOrgList(options: { registry?: string } = {}): Promise<void> {
  const credentials = readCredentials();
  const me = await fetchMe(options.registry);
  const organizations = Array.isArray(me.organizations) ? me.organizations : [];
  if (organizations.length === 0) {
    console.log('No organizations. You can still browse public skills and use personal drafts.');
    return;
  }
  for (const organization of organizations) {
    const marker = credentials?.organizationId === organization.id ? '*' : ' ';
    console.log(`${marker} ${organization.slug}\t${organization.role || ''}`);
  }
}

export async function runOrgUse(slug: string, options: { registry?: string } = {}): Promise<void> {
  const credentials = readCredentials();
  if (!credentials) throw new Error('Run `skills login` first.');
  const me = await fetchMe(options.registry);
  const organizations = Array.isArray(me.organizations) ? me.organizations : [];
  const organization = organizations.find((item: { slug?: string }) => item.slug === slug);
  if (!organization) throw new Error(`Organization not found: ${slug}`);
  writeCredentials({
    ...credentials,
    organizationId: organization.id,
    organizationSlug: organization.slug,
    accessToken: undefined,
    accessTokenExpiresAt: undefined,
  });
  console.log(`Active organization: ${organization.slug}`);
}

export function runLogout(): void {
  clearCredentials();
  console.log('Signed out.');
}

export async function fetchMe(registry?: string): Promise<Record<string, unknown>> {
  const token = await getStoredAccessToken();
  if (!token) throw new Error('Run `skills login` first, or set SKILLS_REGISTRY_TOKEN.');
  const base = getRegistryBaseUrl(registry);
  const response = await fetch(`${base}/api/v1/me`, {
    headers: { authorization: `Bearer ${token}`, accept: 'application/json' },
  });
  if (!response.ok) {
    throw new Error(`Catalog /me failed with ${response.status}.`);
  }
  return (await response.json()) as Record<string, unknown>;
}

async function requestJSON(url: string, init: RequestInit): Promise<Record<string, unknown>> {
  const response = await fetch(url, init);
  let payload: Record<string, unknown> = {};
  try {
    payload = (await response.json()) as Record<string, unknown>;
  } catch {
    payload = {};
  }
  if (!response.ok && !payload.error) {
    throw new Error(`Request failed with ${response.status}.`);
  }
  return payload;
}

function sleep(milliseconds: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
