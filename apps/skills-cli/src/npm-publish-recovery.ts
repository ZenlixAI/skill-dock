import { resolveNpmAuthHeaders } from './npm-auth.ts';
import { normalizeRegistryUrl } from './registry-url.ts';

interface NpmPackageMetadata {
  versions?: Record<string, unknown>;
}

interface EnsurePublishedOptions {
  directory: string;
  packageName: string;
  version: string;
  npmRegistry?: string;
  publish: () => void;
}

function buildPackageMetadataUrl(registryUrl: string, packageName: string): string {
  return `${normalizeRegistryUrl(registryUrl)}/${encodeURIComponent(packageName)}`;
}

function isNpmPackageMetadata(value: unknown): value is NpmPackageMetadata {
  return !!value && typeof value === 'object';
}

async function packageVersionExistsInNpmRegistry(
  packageName: string,
  version: string,
  registryUrl: string,
  directory: string
): Promise<boolean> {
  const metadataUrl = buildPackageMetadataUrl(registryUrl, packageName);
  const authHeaders = resolveNpmAuthHeaders(metadataUrl, { cwd: directory }).headers;
  const response = await fetch(metadataUrl, {
    method: 'GET',
    headers: {
      accept: 'application/json',
      ...authHeaders,
    },
  });

  if (response.status === 404) {
    return false;
  }
  if (!response.ok) {
    throw new Error(`npm registry metadata lookup failed with status ${response.status}`);
  }

  const payload = (await response.json()) as unknown;
  if (!isNpmPackageMetadata(payload)) {
    throw new Error('npm registry metadata response was malformed');
  }

  return Object.prototype.hasOwnProperty.call(payload.versions || {}, version);
}

function isVersionAlreadyPublishedError(error: unknown): boolean {
  if (!(error instanceof Error)) {
    return false;
  }

  const message = error.message.toLowerCase();
  return [
    'cannot publish over the previously published versions',
    'previously published versions',
    'cannot modify pre-existing version',
    'cannot publish over existing version',
  ].some((pattern) => message.includes(pattern));
}

export async function ensurePublishedVersion(options: EnsurePublishedOptions): Promise<void> {
  const npmRegistry = normalizeRegistryUrl(options.npmRegistry);

  if (npmRegistry) {
    const versionAlreadyExists = await packageVersionExistsInNpmRegistry(
      options.packageName,
      options.version,
      npmRegistry,
      options.directory
    );
    if (versionAlreadyExists) {
      return;
    }
  }

  try {
    options.publish();
    return;
  } catch (error) {
    if (!npmRegistry || !isVersionAlreadyPublishedError(error)) {
      throw error;
    }

    const versionAlreadyExists = await packageVersionExistsInNpmRegistry(
      options.packageName,
      options.version,
      npmRegistry,
      options.directory
    );
    if (versionAlreadyExists) {
      return;
    }

    throw error;
  }
}
