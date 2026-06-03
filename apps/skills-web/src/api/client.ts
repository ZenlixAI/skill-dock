import { registryClientConfig } from './config';
import { requestJSON } from './httpClient';
import { mockClient } from './mockClient';
import type {
  AuditEventRecord,
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

const realClient: RegistryApiClient = {
  getConfig() {
    return registryClientConfig;
  },

  async listSkills(filters?: ListSkillsFilters) {
    const response = await requestJSON<SkillsPage<SkillSummary>>(
      withQuery('/api/v1/skills', {
        q: filters?.query?.trim() || undefined,
        source_type: filters?.sourceType || undefined,
        include_retired: filters?.includeRetired || filters?.status === 'retired' ? true : undefined,
        pageSize: 100,
      })
    );
    return filters?.status ? response.items.filter((item) => item.status === filters.status) : response.items;
  },

  async listCuratedSkills() {
    const response = await requestJSON<SkillsPage<CuratedSkillSummary>>(
      withQuery('/api/v1/skills/curated', {
        pageSize: 20,
      })
    );
    return response.items;
  },

  getSkillDetail(packageName: string) {
    return requestJSON<SkillDetail>(`/api/v1/skills/${encodeURIComponent(packageName)}`);
  },

  async getSkillVersions(packageName: string) {
    const response = await requestJSON<SkillsPage<SkillVersionSummary>>(
      withQuery(`/api/v1/skills/${encodeURIComponent(packageName)}/versions`, {
        include_deprecated: true,
        include_yanked: true,
        pageSize: 100,
      })
    );
    return response.items;
  },

  getInstallTarget(packageName: string, version?: string) {
    return requestJSON<InstallTarget>(
      withQuery(`/api/v1/skills/${encodeURIComponent(packageName)}/install-target`, {
        version,
      })
    );
  },

  getVersionDetail(packageName: string, version: string) {
    return requestJSON<SkillVersionDetail>(
      `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${encodeURIComponent(version)}`
    );
  },

  async listGovernanceChanges(filters?: ListGovernanceFilters) {
    const response = await requestJSON<SkillsPage<GovernanceRecord>>(
      withQuery('/api/v1/governance-changes', {
        package_name: filters?.packageName,
        version: filters?.version,
        target_type: filters?.targetType || undefined,
        pageSize: 100,
      })
    );
    return response.items;
  },

  async listAuditEvents(filters?: ListAuditFilters) {
    const response = await requestJSON<SkillsPage<AuditEventRecord>>(
      withQuery('/api/v1/audit-events', {
        action: filters?.action,
        actor: filters?.actor,
        auth_decision: filters?.authDecision,
        package_name: filters?.packageName,
        version: filters?.version,
        outcome: filters?.outcome,
        pageSize: 100,
      })
    );
    return response.items;
  },

  preparePackaging(request: PackagingPrepareRequest) {
    return requestJSON<PackagingPrepareResponse>('/api/v1/skill-packaging/prepare', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  },

  validatePackaging(request: PackagingValidateRequest) {
    return requestJSON<PackagingValidateResponse>('/api/v1/skill-packaging/validate', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  },

  packSkill(request: PackagingPackRequest) {
    return requestJSON<PackagingPackResponse>('/api/v1/skill-packaging/pack', {
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
export const createReleaseRecord = apiClient.createReleaseRecord.bind(apiClient);
export const applySkillGovernanceChange = apiClient.applySkillGovernanceChange.bind(apiClient);
export const applyVersionGovernanceChange = apiClient.applyVersionGovernanceChange.bind(apiClient);
