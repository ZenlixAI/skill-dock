export interface CatalogSkillSummary {
  packageName: string;
  displayName: string;
  description: string;
  latestVersion?: string;
}

export interface CatalogPageInfo {
  nextCursor: string | null;
  hasNextPage: boolean;
}

export interface CatalogSkillsPage {
  items: CatalogSkillSummary[];
  pageInfo: CatalogPageInfo;
}

export interface InstallTarget {
  packageName: string;
  version: string;
  npmRegistry: string;
  tarballUrl: string;
  integrity: string;
  skillManifestDigest: string;
  versionStatus: string;
}

export interface PackagingSource {
  type: 'local' | 'git' | 'platform_authoring';
  location?: string;
  ref?: string;
  files?: Record<string, string>;
}

export interface PackagingTarget {
  packageName: string;
  version: string;
}

export interface PackagingMetadata {
  displayName?: string;
  description?: string;
  sourceType?: string;
}

export interface PackagingOptions {
  publishRegistry?: string;
  strict?: boolean;
}

export interface PackagingWarningDetail {
  code: string;
  severity?: string;
  title: string;
  message: string;
  suggestedAction?: string;
}

export interface PackagingImportReport {
  sourceType?: string;
  strictMode: boolean;
  normalized: boolean;
  generatedPackageJson: boolean;
  generatedCapabilityPlaceholder: boolean;
  rewrotePackageIdentity: boolean;
  generatedPaths?: string[];
  warningCodes?: string[];
}

export interface PreparePackagingResponse {
  workspacePath: string;
  packageName: string;
  version: string;
  warnings: string[];
  warningDetails?: PackagingWarningDetail[];
  importReport?: PackagingImportReport;
  errors: string[];
}

export interface ValidatePackagingResponse extends PreparePackagingResponse {
  ok: boolean;
  manifestDigest?: string;
  components?: string[];
}

export interface PackPackagingResponse extends PreparePackagingResponse {
  bundlePath: string;
  bundleBase64?: string;
  manifestDigest: string;
  components: string[];
}

interface CatalogErrorBody {
  error?: {
    code?: string;
    message?: string;
    details?: Record<string, unknown>;
    requestId?: string;
  };
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string');
}

function isWarningDetailsArray(value: unknown): value is PackagingWarningDetail[] {
  return (
    Array.isArray(value) &&
    value.every(
      (item) =>
        item &&
        typeof item === 'object' &&
        typeof (item as PackagingWarningDetail).code === 'string' &&
        ((item as PackagingWarningDetail).severity === undefined ||
          typeof (item as PackagingWarningDetail).severity === 'string') &&
        typeof (item as PackagingWarningDetail).title === 'string' &&
        typeof (item as PackagingWarningDetail).message === 'string' &&
        ((item as PackagingWarningDetail).suggestedAction === undefined ||
          typeof (item as PackagingWarningDetail).suggestedAction === 'string')
    )
  );
}

function isImportReport(value: unknown): value is PackagingImportReport {
  return (
    !!value &&
    typeof value === 'object' &&
    ((value as PackagingImportReport).sourceType === undefined ||
      typeof (value as PackagingImportReport).sourceType === 'string') &&
    typeof (value as PackagingImportReport).strictMode === 'boolean' &&
    typeof (value as PackagingImportReport).normalized === 'boolean' &&
    typeof (value as PackagingImportReport).generatedPackageJson === 'boolean' &&
    typeof (value as PackagingImportReport).generatedCapabilityPlaceholder === 'boolean' &&
    typeof (value as PackagingImportReport).rewrotePackageIdentity === 'boolean' &&
    ((value as PackagingImportReport).generatedPaths === undefined ||
      isStringArray((value as PackagingImportReport).generatedPaths)) &&
    ((value as PackagingImportReport).warningCodes === undefined ||
      isStringArray((value as PackagingImportReport).warningCodes))
  );
}

export class CatalogError extends Error {
  code: string;
  details?: Record<string, unknown>;
  requestId?: string;
  status: number;

  constructor(params: {
    code: string;
    message: string;
    status: number;
    details?: Record<string, unknown>;
    requestId?: string;
  }) {
    super(params.message);
    this.name = 'CatalogError';
    this.code = params.code;
    this.details = params.details;
    this.requestId = params.requestId;
    this.status = params.status;
  }
}

export function getRegistryBaseUrl(explicit?: string): string {
  const raw = explicit || process.env.SKILLS_REGISTRY_URL || 'https://skills.sh';

  return raw.replace(/\/+$/, '');
}

function getCatalogAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {};
  const token = process.env.SKILLS_REGISTRY_TOKEN?.trim();
  const role = process.env.SKILLS_REGISTRY_ROLE?.trim();

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  if (role) {
    headers['X-Role'] = role;
  }

  return headers;
}

function buildCatalogUrl(
  path: string,
  query?: Record<string, string | undefined>,
  registry?: string
): string {
  const url = new URL(`${getRegistryBaseUrl(registry)}${path}`);

  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== '') {
        url.searchParams.set(key, value);
      }
    }
  }

  return url.toString();
}

async function parseCatalogError(response: Response): Promise<CatalogError> {
  let payload: CatalogErrorBody | undefined;

  try {
    payload = (await response.json()) as CatalogErrorBody;
  } catch {
    payload = undefined;
  }

  return new CatalogError({
    code: payload?.error?.code || 'catalog_request_failed',
    message: payload?.error?.message || `Catalog request failed with status ${response.status}`,
    details: payload?.error?.details,
    requestId: payload?.error?.requestId,
    status: response.status,
  });
}

export async function searchCatalogSkills(
  query: string,
  options: { registry?: string; pageSize?: number } = {}
): Promise<CatalogSkillSummary[]> {
  const page = await searchCatalogSkillsPage(query, options);
  return page.items;
}

export async function searchCatalogSkillsPage(
  query: string,
  options: { registry?: string; pageSize?: number; cursor?: string } = {}
): Promise<CatalogSkillsPage> {
  const url = buildCatalogUrl(
    '/api/v1/skills',
    {
      q: query,
      pageSize: String(options.pageSize ?? 10),
      cursor: options.cursor,
      sort: query ? 'relevance' : 'updated_at_desc',
    },
    options.registry
  );

  const response = await fetch(url, {
    headers: {
      accept: 'application/json',
      ...getCatalogAuthHeaders(),
    },
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }

  const payload = (await response.json()) as {
    items?: Array<{
      packageName: string;
      displayName?: string;
      description?: string;
      latestVersion?: string;
    }>;
    pageInfo?: {
      nextCursor?: string | null;
      hasNextPage?: boolean;
    };
  };

  return {
    items: (payload.items || []).map((item) => ({
      packageName: item.packageName,
      displayName: item.displayName || item.packageName,
      description: item.description || '',
      latestVersion: item.latestVersion,
    })),
    pageInfo: {
      nextCursor: payload.pageInfo?.nextCursor ?? null,
      hasNextPage: payload.pageInfo?.hasNextPage ?? false,
    },
  };
}

export async function putReleaseRecord(
  packageName: string,
  version: string,
  options: {
    registry?: string;
    publishRegistry?: string;
    manifestDigest?: string;
    components?: string[];
  } = {}
): Promise<void> {
  const url = buildCatalogUrl(
    `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${encodeURIComponent(version)}/release-record`,
    undefined,
    options.registry
  );

  const body: Record<string, unknown> = {};
  if (options.publishRegistry) {
    body.publishRegistry = options.publishRegistry;
  }
  if (options.manifestDigest) {
    body.manifestDigest = options.manifestDigest;
  }
  if (options.components && options.components.length > 0) {
    body.components = options.components;
  }

  const response = await fetch(url, {
    method: 'PUT',
    headers: {
      accept: 'application/json',
      'content-type': 'application/json',
      ...getCatalogAuthHeaders(),
    },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }
}

export async function resolveInstallTarget(
  packageName: string,
  options: { registry?: string; version?: string } = {}
): Promise<InstallTarget> {
  const url = buildCatalogUrl(
    `/api/v1/skills/${encodeURIComponent(packageName)}/install-target`,
    { version: options.version },
    options.registry
  );

  const response = await fetch(url, {
    headers: {
      accept: 'application/json',
      ...getCatalogAuthHeaders(),
    },
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }

  return (await response.json()) as InstallTarget;
}

export async function prepareSkillPackaging(input: {
  registry?: string;
  idempotencyKey?: string;
  source: PackagingSource;
  target: PackagingTarget;
  metadata?: PackagingMetadata;
  options?: PackagingOptions;
}): Promise<PreparePackagingResponse> {
  const url = buildCatalogUrl('/api/v1/skill-packaging/prepare', undefined, input.registry);
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      accept: 'application/json',
      'content-type': 'application/json',
      ...getCatalogAuthHeaders(),
    },
    body: JSON.stringify({
      idempotencyKey: input.idempotencyKey,
      source: input.source,
      target: input.target,
      metadata: input.metadata || {},
      options: input.options || {},
    }),
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }

  return (await response.json()) as PreparePackagingResponse;
}

export async function validateSkillPackaging(input: {
  registry?: string;
  idempotencyKey?: string;
  workspacePath: string;
  target: PackagingTarget;
}): Promise<ValidatePackagingResponse> {
  const url = buildCatalogUrl('/api/v1/skill-packaging/validate', undefined, input.registry);
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      accept: 'application/json',
      'content-type': 'application/json',
      ...getCatalogAuthHeaders(),
    },
    body: JSON.stringify({
      idempotencyKey: input.idempotencyKey,
      workspacePath: input.workspacePath,
      target: input.target,
    }),
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }

  return (await response.json()) as ValidatePackagingResponse;
}

export async function packSkillPackaging(input: {
  registry?: string;
  idempotencyKey?: string;
  workspacePath: string;
  target: PackagingTarget;
  includeBundle?: boolean;
}): Promise<PackPackagingResponse> {
  const url = buildCatalogUrl('/api/v1/skill-packaging/pack', undefined, input.registry);
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      accept: 'application/json',
      'content-type': 'application/json',
      ...getCatalogAuthHeaders(),
    },
    body: JSON.stringify({
      idempotencyKey: input.idempotencyKey,
      workspacePath: input.workspacePath,
      target: input.target,
      includeBundle: input.includeBundle,
    }),
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }

  return (await response.json()) as PackPackagingResponse;
}

export async function downloadPackedBundle(input: {
  registry?: string;
  workspacePath: string;
  target: PackagingTarget;
}): Promise<Buffer> {
  const url = buildCatalogUrl(
    '/api/v1/skill-packaging/bundle',
    {
      workspacePath: input.workspacePath,
      packageName: input.target.packageName,
      version: input.target.version,
    },
    input.registry
  );
  const response = await fetch(url, {
    method: 'GET',
    headers: {
      accept: 'application/octet-stream',
      ...getCatalogAuthHeaders(),
    },
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }

  return Buffer.from(await response.arrayBuffer());
}
