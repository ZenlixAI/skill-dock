export type SkillStatus = 'active' | 'retired';
export type VersionStatus = 'published' | 'deprecated' | 'yanked';
export type InstallPolicy = 'catalog_only' | 'resolvable' | 'blocked';
export type CatalogSourceType = 'native' | 'imported' | 'platform_created';
export type PackagingSourceType = 'git' | 'upload_archive' | 'platform_authoring';
export type GovernanceTargetType = 'skill' | 'version';
export type GovernanceAction = 'retire' | 'deprecate' | 'yank';
export type WarningSeverity = 'info' | 'warning' | 'high-risk';
export type RegistryRuntimeMode = 'mock' | 'real';
export type RegistryRequestErrorKind = 'auth' | 'network' | 'not_found' | 'server' | 'unknown';

export interface RegistryRequestError {
  kind: RegistryRequestErrorKind;
  message: string;
  requestId?: string;
}

export interface PageInfo {
  nextCursor?: string | null;
  hasNextPage: boolean;
}

export interface SkillsPage<T> {
  items: T[];
  pageInfo: PageInfo;
}

export interface SkillSummary {
  packageName: string;
  displayName: string;
  description: string;
  sourceType: CatalogSourceType;
  status: SkillStatus;
  latestVersion?: string | null;
  latestVersionStatus?: VersionStatus | null;
  updatedAt: string;
}

export interface CuratedSkillSummary extends SkillSummary {
  curationReason: string;
}

export interface SkillDetail extends SkillSummary {
  createdAt: string;
}

export interface SkillVersionSummary {
  packageName: string;
  version: string;
  versionStatus: VersionStatus;
  installPolicy: InstallPolicy;
  publishedAt: string;
}

export interface RegistryMetadata {
  npmRegistry: string;
  tarballUrl: string;
  integrity: string;
  shasum: string;
  publishedAt: string;
  distTags: string[];
}

export interface BundleMetadata {
  skillManifestDigest: string;
  components: string[];
}

export interface InstallTarget {
  packageName: string;
  version: string;
  npmRegistry: string;
  tarballUrl: string;
  integrity: string;
  skillManifestDigest: string;
  versionStatus: VersionStatus;
}

export interface SkillVersionDetail extends SkillVersionSummary {
  registry: RegistryMetadata;
  bundle: BundleMetadata;
}

export interface GovernanceRecord {
  id: string;
  targetType: GovernanceTargetType;
  packageName: string;
  version?: string;
  action: GovernanceAction;
  reason: string;
  replacementRecommendation?: string;
  changedBy: string;
  changedAt: string;
}

export interface GovernanceChangeResultingState {
  status?: SkillStatus;
  versionStatus?: VersionStatus;
}

export interface GovernanceChangeResponse {
  record: GovernanceRecord;
  resultingState: GovernanceChangeResultingState;
}

export interface AuditEventRecord {
  id: string;
  requestId: string;
  traceId: string;
  action: string;
  actor: string;
  role: string;
  authMechanism: string;
  authDecision: string;
  endpoint: string;
  method: string;
  packageName: string;
  version?: string;
  outcome: string;
  statusCode: number;
  details: Record<string, unknown>;
  createdAt: string;
}

export interface WarningDetail {
  code: string;
  severity: WarningSeverity;
  title: string;
  message: string;
  suggestedAction?: string;
}

export interface ImportReport {
  sourceType?: string;
  strictMode: boolean;
  normalized: boolean;
  generatedPackageJson: boolean;
  generatedCapabilityPlaceholder: boolean;
  rewrotePackageIdentity: boolean;
  generatedPaths?: string[];
  warningCodes?: string[];
}

export interface PackagingSource {
  type: PackagingSourceType;
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
  displayName: string;
  description: string;
  sourceType?: CatalogSourceType | string;
}

export interface PackagingOptions {
  publishRegistry?: string;
  strict?: boolean;
}

export interface PackagingPrepareRequest {
  source: PackagingSource;
  target: PackagingTarget;
  metadata: PackagingMetadata;
  options: PackagingOptions;
  idempotencyKey?: string;
}

export interface PackagingPrepareResponse {
  workspacePath: string;
  packageName: string;
  version: string;
  warnings: string[];
  warningDetails: WarningDetail[];
  importReport?: ImportReport;
  errors: string[];
}

export interface PackagingValidateRequest {
  workspacePath: string;
  target: PackagingTarget;
  idempotencyKey?: string;
}

export interface PackagingValidateResponse {
  ok: boolean;
  workspacePath: string;
  packageName: string;
  version: string;
  manifestDigest?: string;
  components: string[];
  warnings: string[];
  warningDetails: WarningDetail[];
  importReport?: ImportReport;
  errors: string[];
}

export interface PackagingPackRequest {
  workspacePath: string;
  target: PackagingTarget;
  includeBundle?: boolean;
  idempotencyKey?: string;
}

export interface PackagingPackResponse {
	workspacePath: string;
	bundlePath: string;
  bundleBase64?: string;
  packageName: string;
  version: string;
  manifestDigest: string;
  components: string[];
  warnings: string[];
  warningDetails: WarningDetail[];
	importReport?: ImportReport;
	errors: string[];
}

export interface PackagingPublishRequest {
  workspacePath: string;
  target: PackagingTarget;
}

export interface PackagingPublishResponse {
  workspacePath: string;
  packageName: string;
  version: string;
  manifestDigest: string;
  components: string[];
  publishedToRegistry: boolean;
  created: boolean;
  idempotentReplay: boolean;
  versionStatus: VersionStatus;
  installPolicy: InstallPolicy;
  warnings: string[];
  warningDetails: WarningDetail[];
  importReport?: ImportReport;
  errors: string[];
}

export interface PackagingCleanupRequest {
  workspacePath: string;
}

export interface PackagingCleanupResponse {
  workspacePath: string;
  removed: boolean;
}

export interface ReleaseRecordInput {
  publishRegistry?: string;
  manifestDigest?: string;
  components?: string[];
}

export interface ReleaseRecordResult {
  packageName: string;
  version: string;
  created: boolean;
  idempotentReplay: boolean;
  versionStatus: VersionStatus;
  installPolicy: InstallPolicy;
}

export interface SkillGovernanceChangeInput {
  action: Extract<GovernanceAction, 'retire'>;
  reason: string;
}

export interface VersionGovernanceChangeInput {
  action: Extract<GovernanceAction, 'deprecate' | 'yank'>;
  reason: string;
  replacementRecommendation?: string;
}

export interface ListSkillsFilters {
  query?: string;
  sourceType?: CatalogSourceType | '';
  status?: SkillStatus | '';
  includeRetired?: boolean;
  cursor?: string;
  pageSize?: number;
}

export interface ListGovernanceFilters {
  packageName?: string;
  version?: string;
  targetType?: GovernanceTargetType | '';
  cursor?: string;
  pageSize?: number;
}

export interface ListAuditFilters {
  action?: string;
  actor?: string;
  authDecision?: string;
  packageName?: string;
  version?: string;
  outcome?: string;
  cursor?: string;
  pageSize?: number;
}

export interface RegistrySession {
  role: 'reader' | 'authoring' | 'publisher' | 'operator';
  actor: string;
  authMechanism: string;
  allowedActions: string[];
  allowedPackages: string[];
}

export interface RegistryClientConfig {
  mode: RegistryRuntimeMode;
  baseUrl: string | null;
  role: string | null;
  actor: string | null;
  hasToken: boolean;
}

export interface RegistryApiClient {
  getConfig(): RegistryClientConfig;
  getSession(): Promise<RegistrySession | null>;
  loginSession(username: string, password: string): Promise<RegistrySession>;
  logoutSession(): Promise<void>;
  listSkills(filters?: ListSkillsFilters): Promise<SkillsPage<SkillSummary>>;
  listCuratedSkills(): Promise<CuratedSkillSummary[]>;
  getSkillDetail(packageName: string): Promise<SkillDetail>;
  getSkillVersions(
    packageName: string,
    options?: {
      cursor?: string;
      pageSize?: number;
    }
  ): Promise<SkillsPage<SkillVersionSummary>>;
  getInstallTarget(packageName: string, version?: string): Promise<InstallTarget>;
  getVersionDetail(packageName: string, version: string): Promise<SkillVersionDetail>;
  listGovernanceChanges(filters?: ListGovernanceFilters): Promise<SkillsPage<GovernanceRecord>>;
  listAuditEvents(filters?: ListAuditFilters): Promise<SkillsPage<AuditEventRecord>>;
  preparePackaging(request: PackagingPrepareRequest): Promise<PackagingPrepareResponse>;
  validatePackaging(request: PackagingValidateRequest): Promise<PackagingValidateResponse>;
  packSkill(request: PackagingPackRequest): Promise<PackagingPackResponse>;
  publishSkill(request: PackagingPublishRequest): Promise<PackagingPublishResponse>;
  cleanupPackagingWorkspace(request: PackagingCleanupRequest): Promise<PackagingCleanupResponse>;
  createReleaseRecord(packageName: string, version: string, input: ReleaseRecordInput): Promise<ReleaseRecordResult>;
  applySkillGovernanceChange(packageName: string, input: SkillGovernanceChangeInput): Promise<GovernanceChangeResponse>;
  applyVersionGovernanceChange(
    packageName: string,
    version: string,
    input: VersionGovernanceChangeInput
  ): Promise<GovernanceChangeResponse>;
}
