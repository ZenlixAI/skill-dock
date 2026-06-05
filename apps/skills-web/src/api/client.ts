import { registryClientConfig } from './config';
import { RegistryHttpError, requestJSON } from './httpClient';
import { mockClient } from './mockClient';
import type {
  AuditEventRecord,
  BundleMetadata,
  PackagingCleanupRequest,
  PackagingCleanupResponse,
  CuratedSkillSummary,
  GovernanceChangeResponse,
  GovernanceRecord,
  InstallTarget,
  ListAuditFilters,
  ListGovernanceFilters,
  ListSkillsFilters,
  PackagingPackRequest,
  PackagingPackResponse,
  PackagingPrepareRequest,
  PackagingPrepareResponse,
  PackagingValidateRequest,
  PackagingValidateResponse,
  RegistryApiClient,
  RegistrySession,
  ReleaseRecordInput,
  ReleaseRecordResult,
  SkillDetail,
  SkillGovernanceChangeInput,
  SkillsPage,
  SkillSummary,
  SkillVersionDetail,
  SkillVersionSummary,
  VersionGovernanceChangeInput,
} from './types';

function withQuery(path: string, params: Record<string, string | number | boolean | undefined>) {
  const url = new URL(path, 'http://registry.local');
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === '' || value === false) {
      return;
    }
    url.searchParams.set(key, String(value));
  });
  return `${url.pathname}${url.search}`;
}

export function normalizeSkillsPage<T>(response: {
  items?: T[] | null;
  pageInfo?: Partial<SkillsPage<T>['pageInfo']> | null;
} | null | undefined): SkillsPage<T> {
  return {
    items: Array.isArray(response?.items) ? response.items : [],
    pageInfo: {
      hasNextPage: response?.pageInfo?.hasNextPage ?? false,
      nextCursor: response?.pageInfo?.nextCursor ?? null,
    },
  };
}

export function normalizeBundleMetadata(bundle: Partial<BundleMetadata> | null | undefined): BundleMetadata {
  return {
    skillManifestDigest: bundle?.skillManifestDigest || '',
    components: Array.isArray(bundle?.components) ? bundle.components : [],
  };
}

export function normalizePackagingReview<
  T extends { warnings?: string[] | null; warningDetails?: unknown[] | null; errors?: string[] | null },
>(
  response: T
) {
  return {
    ...response,
    warnings: Array.isArray(response.warnings) ? response.warnings : [],
    warningDetails: Array.isArray(response.warningDetails) ? response.warningDetails : [],
    errors: Array.isArray(response.errors) ? response.errors : [],
  };
}

const realClient: RegistryApiClient = {
  getConfig() {
    return registryClientConfig;
  },

  getSession() {
    return requestJSON<RegistrySession | null>('/api/v1/session').catch((error: unknown) => {
      if (error instanceof RegistryHttpError && error.status === 401) {
        return null;
      }
      throw error;
    });
  },

  loginSession(username: string, password: string) {
    return requestJSON<RegistrySession>('/api/v1/session/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  },

  async logoutSession() {
    await requestJSON<{ ok: boolean }>('/api/v1/session/logout', {
      method: 'POST',
      body: JSON.stringify({}),
    });
  },

  async listSkills(filters?: ListSkillsFilters) {
    const response = normalizeSkillsPage(
      await requestJSON<SkillsPage<SkillSummary>>(
        withQuery('/api/v1/skills', {
          q: filters?.query?.trim() || undefined,
          source_type: filters?.sourceType || undefined,
          include_retired: filters?.includeRetired || filters?.status === 'retired' ? true : undefined,
          cursor: filters?.cursor,
          pageSize: filters?.pageSize ?? 12,
        })
      )
    );
    return {
      items: filters?.status ? response.items.filter((item) => item.status === filters.status) : response.items,
      pageInfo: response.pageInfo,
    };
  },

  async listCuratedSkills() {
    const response = normalizeSkillsPage(
      await requestJSON<SkillsPage<CuratedSkillSummary>>(
        withQuery('/api/v1/skills/curated', {
          pageSize: 20,
        })
      )
    );
    return response.items;
  },

  getSkillDetail(packageName: string) {
    return requestJSON<SkillDetail>(`/api/v1/skills/${encodeURIComponent(packageName)}`);
  },

  getSkillVersions(packageName: string, options?: { cursor?: string; pageSize?: number }) {
    return requestJSON<SkillsPage<SkillVersionSummary>>(
      withQuery(`/api/v1/skills/${encodeURIComponent(packageName)}/versions`, {
        include_deprecated: true,
        include_yanked: true,
        cursor: options?.cursor,
        pageSize: options?.pageSize ?? 20,
      })
    ).then(normalizeSkillsPage);
  },

  getInstallTarget(packageName: string, version?: string) {
    return requestJSON<InstallTarget>(
      withQuery(`/api/v1/skills/${encodeURIComponent(packageName)}/install-target`, {
        version,
      })
    );
  },

  async getVersionDetail(packageName: string, version: string) {
    const response = await requestJSON<SkillVersionDetail>(
      `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${encodeURIComponent(version)}`
    );
    return {
      ...response,
      bundle: normalizeBundleMetadata(response.bundle),
    };
  },

  async listGovernanceChanges(filters?: ListGovernanceFilters) {
    return requestJSON<SkillsPage<GovernanceRecord>>(
      withQuery('/api/v1/governance-changes', {
        package_name: filters?.packageName,
        version: filters?.version,
        target_type: filters?.targetType || undefined,
        cursor: filters?.cursor,
        pageSize: filters?.pageSize ?? 10,
      })
    ).then(normalizeSkillsPage);
  },

  async listAuditEvents(filters?: ListAuditFilters) {
    return requestJSON<SkillsPage<AuditEventRecord>>(
      withQuery('/api/v1/audit-events', {
        action: filters?.action,
        actor: filters?.actor,
        auth_decision: filters?.authDecision,
        package_name: filters?.packageName,
        version: filters?.version,
        outcome: filters?.outcome,
        cursor: filters?.cursor,
        pageSize: filters?.pageSize ?? 10,
      })
    ).then(normalizeSkillsPage);
  },

  async preparePackaging(request: PackagingPrepareRequest) {
    const response = await requestJSON<PackagingPrepareResponse>('/api/v1/skill-packaging/prepare', {
      method: 'POST',
      body: JSON.stringify(request),
    });
    return normalizePackagingReview(response);
  },

  async validatePackaging(request: PackagingValidateRequest) {
    const response = await requestJSON<PackagingValidateResponse>('/api/v1/skill-packaging/validate', {
      method: 'POST',
      body: JSON.stringify(request),
    });
    return {
      ...normalizePackagingReview(response),
      components: Array.isArray(response.components) ? response.components : [],
    };
  },

  async packSkill(request: PackagingPackRequest) {
    const response = await requestJSON<PackagingPackResponse>('/api/v1/skill-packaging/pack', {
      method: 'POST',
      body: JSON.stringify(request),
    });
    return {
      ...normalizePackagingReview(response),
      components: Array.isArray(response.components) ? response.components : [],
    };
  },

  cleanupPackagingWorkspace(request: PackagingCleanupRequest) {
    return requestJSON<PackagingCleanupResponse>('/api/v1/skill-packaging/cleanup', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  },

  createReleaseRecord(packageName: string, version: string, input: ReleaseRecordInput) {
    return requestJSON<ReleaseRecordResult>(
      `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${encodeURIComponent(version)}/release-record`,
      {
        method: 'PUT',
        body: JSON.stringify(input),
      }
    );
  },

  applySkillGovernanceChange(packageName: string, input: SkillGovernanceChangeInput) {
    return requestJSON<GovernanceChangeResponse>(
      `/api/v1/skills/${encodeURIComponent(packageName)}/governance-changes`,
      {
        method: 'POST',
        body: JSON.stringify(input),
      }
    );
  },

  applyVersionGovernanceChange(packageName: string, version: string, input: VersionGovernanceChangeInput) {
    return requestJSON<GovernanceChangeResponse>(
      `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${encodeURIComponent(version)}/governance-changes`,
      {
        method: 'POST',
        body: JSON.stringify(input),
      }
    );
  },
};

export const apiClient: RegistryApiClient = registryClientConfig.mode === 'real' ? realClient : mockClient;

export const getRegistryClientConfig = () => apiClient.getConfig();
export const getSession = apiClient.getSession.bind(apiClient);
export const loginSession = apiClient.loginSession.bind(apiClient);
export const logoutSession = apiClient.logoutSession.bind(apiClient);

export const listSkills = apiClient.listSkills.bind(apiClient);
export const listCuratedSkills = apiClient.listCuratedSkills.bind(apiClient);
export const getSkillDetail = apiClient.getSkillDetail.bind(apiClient);
export const getSkillVersions = apiClient.getSkillVersions.bind(apiClient);
export const getInstallTarget = apiClient.getInstallTarget.bind(apiClient);
export const getVersionDetail = apiClient.getVersionDetail.bind(apiClient);
export const listGovernanceChanges = apiClient.listGovernanceChanges.bind(apiClient);
export const listAuditEvents = apiClient.listAuditEvents.bind(apiClient);
export const preparePackaging = apiClient.preparePackaging.bind(apiClient);
export const validatePackaging = apiClient.validatePackaging.bind(apiClient);
export const packSkill = apiClient.packSkill.bind(apiClient);
export const cleanupPackagingWorkspace = apiClient.cleanupPackagingWorkspace.bind(apiClient);
export const createReleaseRecord = apiClient.createReleaseRecord.bind(apiClient);
export const applySkillGovernanceChange = apiClient.applySkillGovernanceChange.bind(apiClient);
export const applyVersionGovernanceChange = apiClient.applyVersionGovernanceChange.bind(apiClient);
