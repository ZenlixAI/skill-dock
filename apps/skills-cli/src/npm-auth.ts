import { existsSync, readFileSync } from 'fs';
import { join, resolve } from 'path';

export interface NpmAuthContextOptions {
  cwd?: string;
  userConfigPath?: string;
  env?: NodeJS.ProcessEnv;
}

export interface NpmAuthHeadersResult {
  headers: Record<string, string>;
  userConfigPath?: string;
}

interface NpmRcEntry {
  key: string;
  value: string;
}

function normalizeRegistryUrl(url: string): URL {
  const normalized = new URL(url);
  normalized.pathname = normalized.pathname.replace(/\/+$/, '') + '/';
  normalized.search = '';
  normalized.hash = '';
  return normalized;
}

function registryLookupKey(url: string): string {
  const normalized = normalizeRegistryUrl(url);
  return `//${normalized.host}${normalized.pathname}`;
}

function readNpmRcEntries(path: string): NpmRcEntry[] {
  const content = readFileSync(path, 'utf-8');
  const entries: NpmRcEntry[] = [];

  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#') || line.startsWith(';')) continue;
    const equalsIndex = line.indexOf('=');
    if (equalsIndex <= 0) continue;

    entries.push({
      key: line.slice(0, equalsIndex).trim(),
      value: line.slice(equalsIndex + 1).trim(),
    });
  }

  return entries;
}

function candidateNpmRcPaths(options: NpmAuthContextOptions = {}): string[] {
  const env = options.env || process.env;
  const candidates = new Set<string>();

  if (options.userConfigPath) {
    candidates.add(resolve(options.userConfigPath));
  }

  if (env.NPM_CONFIG_USERCONFIG) {
    candidates.add(resolve(env.NPM_CONFIG_USERCONFIG));
  }

  const cwd = resolve(options.cwd || process.cwd());
  candidates.add(join(cwd, '.npmrc'));

  if (env.HOME) {
    candidates.add(join(resolve(env.HOME), '.npmrc'));
  }

  if (env.USERPROFILE) {
    candidates.add(join(resolve(env.USERPROFILE), '.npmrc'));
  }

  return [...candidates].filter((candidate) => existsSync(candidate));
}

function findScopedEntry(entries: NpmRcEntry[], registryUrl: string, suffix: string): string | null {
  const lookupKey = registryLookupKey(registryUrl);
  const scopedEntries = entries
    .filter((entry) => entry.key.endsWith(suffix) && lookupKey.startsWith(entry.key.slice(0, -suffix.length)))
    .sort((left, right) => right.key.length - left.key.length);

  return scopedEntries[0]?.value || null;
}

function decodeBase64Value(value: string): string {
  return Buffer.from(value, 'base64').toString('utf-8');
}

export function resolveNpmAuthHeaders(
  registryUrl: string,
  options: NpmAuthContextOptions = {}
): NpmAuthHeadersResult {
  const paths = candidateNpmRcPaths(options);

  for (const path of paths) {
    const entries = readNpmRcEntries(path);

    const authToken = findScopedEntry(entries, registryUrl, ':_authToken');
    if (authToken) {
      return {
        headers: { Authorization: `Bearer ${authToken}` },
        userConfigPath: path,
      };
    }

    const basicAuth = findScopedEntry(entries, registryUrl, ':_auth');
    if (basicAuth) {
      return {
        headers: { Authorization: `Basic ${basicAuth}` },
        userConfigPath: path,
      };
    }

    const username = findScopedEntry(entries, registryUrl, ':username');
    const encodedPassword = findScopedEntry(entries, registryUrl, ':_password');
    if (username && encodedPassword) {
      const password = decodeBase64Value(encodedPassword);
      const credentials = Buffer.from(`${username}:${password}`).toString('base64');
      return {
        headers: { Authorization: `Basic ${credentials}` },
        userConfigPath: path,
      };
    }
  }

  return { headers: {} };
}

export function resolveNpmUserConfigPath(options: NpmAuthContextOptions = {}): string | undefined {
  return candidateNpmRcPaths(options)[0];
}
