import { chmodSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'fs';
import { homedir } from 'os';
import { dirname, join } from 'path';

export interface StoredCredentials {
  logtoEndpoint: string;
  clientId: string;
  apiResource?: string;
  refreshToken: string;
  accessToken?: string;
  accessTokenExpiresAt?: number;
  organizationId?: string;
  organizationSlug?: string;
}

export interface CredentialsStoreOptions {
  path?: string;
}

export function credentialsPath(options: CredentialsStoreOptions = {}): string {
  if (options.path) return options.path;
  const configHome = process.env.XDG_CONFIG_HOME?.trim() || join(homedir(), '.config');
  return join(configHome, 'skill-dock', 'credentials.json');
}

export function readCredentials(options: CredentialsStoreOptions = {}): StoredCredentials | null {
  const path = credentialsPath(options);
  if (!existsSync(path)) return null;
  try {
    const parsed = JSON.parse(readFileSync(path, 'utf-8')) as StoredCredentials;
    if (!parsed.refreshToken || !parsed.logtoEndpoint || !parsed.clientId) return null;
    return parsed;
  } catch {
    return null;
  }
}

export function writeCredentials(credentials: StoredCredentials, options: CredentialsStoreOptions = {}): void {
  const path = credentialsPath(options);
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, `${JSON.stringify(credentials, null, 2)}\n`, { mode: 0o600 });
  try {
    chmodSync(path, 0o600);
  } catch {
    // Best effort on filesystems without POSIX modes.
  }
}

export function clearCredentials(options: CredentialsStoreOptions = {}): void {
  const path = credentialsPath(options);
  if (existsSync(path)) rmSync(path, { force: true });
}

export function logtoEndpoint(): string {
  return (process.env.SKILLS_LOGTO_ENDPOINT || '').trim().replace(/\/+$/, '');
}

export function cliClientId(): string {
  return (process.env.SKILLS_LOGTO_CLIENT_ID || '').trim();
}

export function apiResource(): string {
  return (process.env.SKILLS_LOGTO_API_RESOURCE || '').trim();
}
