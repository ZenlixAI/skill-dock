import { requestJSON } from './httpClient';
import type {
  AuditEventRecord,
  BundleMetadata,
  CuratedSkillSummary,
  GovernanceRecord,
  InstallTarget,
  ListAuditFilters,
  ListGovernanceFilters,
  ListSkillsFilters,
  RegistryApiClient,
  SkillDetail,
  SkillsPage,
  SkillSummary,
  SkillVersionDetail,
  SkillVersionSummary,
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

export function normalizeSkillsPage<T>(
  response:
    | {
        items?: T[] | null;
        pageInfo?: Partial<SkillsPage<T>['pageInfo']> | null;
      }
    | null
    | undefined
): SkillsPage<T> {
  return {
    items: Array.isArray(response?.items) ? response.items : [],
    pageInfo: {
      hasNextPage: response?.pageInfo?.hasNextPage ?? false,
      nextCursor: response?.pageInfo?.nextCursor ?? null,
    },
  };
}

export function normalizeBundleMetadata(
  bundle: Partial<BundleMetadata> | null | undefined
): BundleMetadata {
  return {
    skillManifestDigest: bundle?.skillManifestDigest || '',
    components: Array.isArray(bundle?.components) ? bundle.components : [],
  };
}

export const catalogReadClient: Pick<
  RegistryApiClient,
  | 'listSkills'
  | 'listCuratedSkills'
  | 'getSkillDetail'
  | 'getSkillVersions'
  | 'getInstallTarget'
  | 'getVersionDetail'
  | 'listGovernanceChanges'
  | 'listAuditEvents'
> = {
  async listSkills(filters?: ListSkillsFilters) {
    const response = normalizeSkillsPage(
      await requestJSON<SkillsPage<SkillSummary>>(
        withQuery('/api/v1/skills', {
          q: filters?.query?.trim() || undefined,
          source_type: filters?.sourceType || undefined,
          include_retired:
            filters?.includeRetired || filters?.status === 'retired' ? true : undefined,
          cursor: filters?.cursor,
          pageSize: filters?.pageSize ?? 12,
        })
      )
    );
    return {
      items: filters?.status
        ? response.items.filter((item) => item.status === filters.status)
        : response.items,
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
};
