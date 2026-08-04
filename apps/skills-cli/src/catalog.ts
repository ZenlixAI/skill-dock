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
  type: 'git' | 'platform_authoring' | 'upload_archive';
  location?: string;
  ref?: string;
  files?: Record<string, string>;
  archiveBase64?: string;
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

export interface ImportReviewPackagingResponse {
  ok: boolean;
  packageName: string;
  version: string;
  manifestDigest?: string;
  components?: string[];
  bundleBase64?: string;
  warnings: string[];
  warningDetails?: PackagingWarningDetail[];
  importReport?: PackagingImportReport;
  errors: string[];
}

export interface PackagingPublishResponse {
  packageName: string;
  version: string;
  manifestDigest: string;
  components: string[];
  publishedToRegistry: boolean;
  created: boolean;
  idempotentReplay: boolean;
  versionStatus: string;
  installPolicy: string;
  warnings: string[];
  warningDetails?: PackagingWarningDetail[];
  importReport?: PackagingImportReport;
  errors: string[];
}

export interface PublishReviewSummary {
  manifestDigest?: string;
  components?: string[];
}

interface CatalogErrorBody {
  error?: {
    code?: string;
    message?: string;
    details?: Record<string, unknown>;
    requestId?: string;
  };
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' ? (value as Record<string, unknown>) : null;
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

function readRequiredStringField(payload: Record<string, unknown>, key: string, context: string): string {
  const value = payload[key];
  if (typeof value !== 'string') {
    throw new Error(`${context} response was malformed: ${key} must be a string`);
  }
  return value;
}

function readOptionalStringField(payload: Record<string, unknown>, key: string, context: string): string | undefined {
  const value = payload[key];
  if (value === undefined || value === null) {
    return undefined;
  }
  if (typeof value !== 'string') {
    throw new Error(`${context} response was malformed: ${key} must be a string when present`);
  }
  return value;
}

function normalizeStringArrayField(
  payload: Record<string, unknown>,
  key: string,
  context: string
): string[] {
  const value = payload[key];
  if (value === undefined || value === null) {
    return [];
  }
  if (!isStringArray(value)) {
    throw new Error(`${context} response was malformed: ${key} must be a string array`);
  }
  return value;
}

function normalizeWarningDetailsField(
  payload: Record<string, unknown>,
  key: string,
  context: string
): PackagingWarningDetail[] | undefined {
  const value = payload[key];
  if (value === undefined || value === null) {
    return undefined;
  }
  if (!isWarningDetailsArray(value)) {
    throw new Error(`${context} response was malformed: ${key} must be a warning detail array`);
  }
  return value;
}

function normalizeImportReportField(
  payload: Record<string, unknown>,
  key: string,
  context: string
): PackagingImportReport | undefined {
  const value = payload[key];
  if (value === undefined || value === null) {
    return undefined;
  }
  if (!isImportReport(value)) {
    throw new Error(`${context} response was malformed: ${key} must be a packaging import report`);
  }
  return value;
}

function normalizeImportReviewPackagingResponse(payload: unknown): ImportReviewPackagingResponse {
  const record = asRecord(payload);
  if (!record) {
    throw new Error('catalog import-review response was malformed');
  }

  const ok = record.ok;
  if (typeof ok !== 'boolean') {
    throw new Error('catalog import-review response was malformed: ok must be a boolean');
  }

  return {
    ok,
    packageName: readRequiredStringField(record, 'packageName', 'catalog import-review'),
    version: readRequiredStringField(record, 'version', 'catalog import-review'),
    manifestDigest: readOptionalStringField(record, 'manifestDigest', 'catalog import-review'),
    components:
      record.components === undefined || record.components === null
        ? undefined
        : normalizeStringArrayField(record, 'components', 'catalog import-review'),
    bundleBase64: readOptionalStringField(record, 'bundleBase64', 'catalog import-review'),
    warnings: normalizeStringArrayField(record, 'warnings', 'catalog import-review'),
    warningDetails: normalizeWarningDetailsField(record, 'warningDetails', 'catalog import-review'),
    importReport: normalizeImportReportField(record, 'importReport', 'catalog import-review'),
    errors: normalizeStringArrayField(record, 'errors', 'catalog import-review'),
  };
}

function normalizePublishPackagingResponse(payload: unknown): PackagingPublishResponse {
  const record = asRecord(payload);
  if (!record) {
    throw new Error('catalog publish response was malformed');
  }

  return {
    packageName: readRequiredStringField(record, 'packageName', 'catalog publish'),
    version: readRequiredStringField(record, 'version', 'catalog publish'),
    manifestDigest: readRequiredStringField(record, 'manifestDigest', 'catalog publish'),
    components: normalizeStringArrayField(record, 'components', 'catalog publish'),
    publishedToRegistry: Boolean(record.publishedToRegistry),
    created: Boolean(record.created),
    idempotentReplay: Boolean(record.idempotentReplay),
    versionStatus: readRequiredStringField(record, 'versionStatus', 'catalog publish'),
    installPolicy: readRequiredStringField(record, 'installPolicy', 'catalog publish'),
    warnings: normalizeStringArrayField(record, 'warnings', 'catalog publish'),
    warningDetails: normalizeWarningDetailsField(record, 'warningDetails', 'catalog publish'),
    importReport: normalizeImportReportField(record, 'importReport', 'catalog publish'),
    errors: normalizeStringArrayField(record, 'errors', 'catalog publish'),
  };
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
  const raw = explicit?.trim() || process.env.SKILLS_REGISTRY_URL?.trim();
  if (!raw) {
    throw new Error(
      'A registry URL is required in offline deployments. Set SKILLS_REGISTRY_URL or pass --registry.'
    );
  }

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

export async function importReviewPackaging(input: {
  registry?: string;
  idempotencyKey?: string;
  source: PackagingSource;
  target: PackagingTarget;
  metadata?: PackagingMetadata;
  options?: PackagingOptions;
  includeBundleBase64?: boolean;
}): Promise<ImportReviewPackagingResponse> {
  const url = buildCatalogUrl('/api/v1/skill-packaging/import-review', undefined, input.registry);
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
      includeBundleBase64: input.includeBundleBase64,
    }),
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }

  return normalizeImportReviewPackagingResponse(await response.json());
}

export async function publishSkillPackaging(input: {
  registry?: string;
  idempotencyKey?: string;
  source: PackagingSource;
  target: PackagingTarget;
  metadata?: PackagingMetadata;
  options?: PackagingOptions;
  review?: PublishReviewSummary;
}): Promise<PackagingPublishResponse> {
  const url = buildCatalogUrl('/api/v1/skill-packaging/publish', undefined, input.registry);
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
      review: input.review,
    }),
  });

  if (!response.ok) {
    throw await parseCatalogError(response);
  }

  return normalizePublishPackagingResponse(await response.json());
}
