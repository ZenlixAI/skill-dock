import type {
  GovernanceChangeResponse,
  GovernanceRecord,
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
  RegistryClientConfig,
  ReleaseRecordInput,
  ReleaseRecordResult,
  SkillGovernanceChangeInput,
  SkillVersionSummary,
  VersionGovernanceChangeInput,
} from './types';
import {
  buildInstallTarget,
  buildVersionDetail,
  createMockState,
  curatedSkills,
  delay,
  nextGovernanceId,
  paginateItems,
  previewImportReport,
  previewWarningDetails,
  recordAudit,
  setLatestSkillState,
} from './mockState';

const config: RegistryClientConfig = {
  mode: 'mock',
  baseUrl: null,
  role: null,
  actor: null,
  hasToken: false,
};

const state = createMockState();

function nowISOString() {
  return new Date().toISOString();
}

export const mockClient: RegistryApiClient = {
  getConfig() {
    return config;
  },

  async getSession() {
    return delay(null);
  },

  async loginSession() {
    return delay({
      role: 'operator',
      actor: 'mock-operator',
      authMechanism: 'mock',
      allowedActions: [
        'session.read',
        'catalog.read',
        'audit.read',
        'packaging.prepare',
        'packaging.validate',
        'packaging.pack',
        'catalog.release_record.create',
        'catalog.governance.skill.apply',
        'catalog.governance.version.apply',
      ],
      allowedPackages: [],
    });
  },

  async logoutSession() {
    await delay(undefined);
  },

  async listSkills(filters?: ListSkillsFilters) {
    const query = filters?.query?.trim().toLowerCase() || '';
    const filtered = state.skills.filter((skill) => {
      const matchesQuery =
        query === '' ||
        skill.packageName.toLowerCase().includes(query) ||
        skill.displayName.toLowerCase().includes(query) ||
        skill.description.toLowerCase().includes(query);
      const matchesSource = !filters?.sourceType || skill.sourceType === filters.sourceType;
      const includeRetired = filters?.includeRetired || filters?.status === 'retired';
      const matchesRetired = includeRetired ? true : skill.status !== 'retired';
      const matchesStatus = !filters?.status || skill.status === filters.status;
      return matchesQuery && matchesSource && matchesStatus && matchesRetired;
    });
    return delay(
      paginateItems(filtered, {
        cursor: filters?.cursor,
        pageSize: filters?.pageSize ?? 12,
      })
    );
  },

  async listCuratedSkills() {
    return delay(curatedSkills);
  },

  async getSkillDetail(packageName: string) {
    const skill = state.skills.find((item) => item.packageName === packageName) || state.skills[0];
    return delay({
      ...skill,
      createdAt: '2026-05-20T02:00:00Z',
    });
  },

  async getSkillVersions(packageName: string, options?: { cursor?: string; pageSize?: number }) {
    return delay(
      paginateItems(state.versions[packageName] || state.versions['@platform.skill/pdf-skill'], {
        cursor: options?.cursor,
        pageSize: options?.pageSize ?? 20,
      })
    );
  },

  async getInstallTarget(packageName: string, version?: string) {
    return delay(buildInstallTarget(state, packageName, version));
  },

  async getVersionDetail(packageName: string, version: string) {
    return delay(buildVersionDetail(state, packageName, version));
  },

  async listGovernanceChanges(filters?: ListGovernanceFilters) {
    const filtered = state.governance.filter((item) => {
      if (filters?.packageName && item.packageName !== filters.packageName) {
        return false;
      }
      if (filters?.version && item.version !== filters.version) {
        return false;
      }
      if (filters?.targetType && item.targetType !== filters.targetType) {
        return false;
      }
      return true;
    });
    return delay(
      paginateItems(filtered, {
        cursor: filters?.cursor,
        pageSize: filters?.pageSize ?? 10,
      })
    );
  },

  async listAuditEvents(filters?: ListAuditFilters) {
    const filtered = state.auditEvents.filter((item) => {
      if (filters?.action && item.action !== filters.action) {
        return false;
      }
      if (filters?.actor && item.actor !== filters.actor) {
        return false;
      }
      if (filters?.authDecision && item.authDecision !== filters.authDecision) {
        return false;
      }
      if (filters?.packageName && item.packageName !== filters.packageName) {
        return false;
      }
      if (filters?.version && item.version !== filters.version) {
        return false;
      }
      if (filters?.outcome && item.outcome !== filters.outcome) {
        return false;
      }
      return true;
    });
    return delay(
      paginateItems(filtered, {
        cursor: filters?.cursor,
        pageSize: filters?.pageSize ?? 10,
      })
    );
  },

  async preparePackaging(request: PackagingPrepareRequest) {
    const warningDetails = request.options.strict ? [] : previewWarningDetails;
    const warnings = warningDetails.map((item) => item.code);
    const response: PackagingPrepareResponse = {
      workspacePath: `/tmp/skill-packager/${request.target.packageName.split('/').pop() || 'skill'}`,
      packageName: request.target.packageName,
      version: request.target.version,
      warnings,
      warningDetails,
      importReport: {
        ...previewImportReport,
        sourceType: request.source.type,
        strictMode: Boolean(request.options.strict),
        normalized: !request.options.strict,
        generatedCapabilityPlaceholder: !request.options.strict,
        warningCodes: warnings,
      },
      errors: [],
    };
    recordAudit(state, {
      action: 'packaging.prepare',
      actor: 'web-ui',
      role: 'publisher',
      authMechanism: 'mock',
      authDecision: 'allowed',
      endpoint: '/api/v1/skill-packaging/prepare',
      method: 'POST',
      packageName: request.target.packageName,
      version: request.target.version,
      outcome: 'succeeded',
      statusCode: 200,
      details: {
        sourceType: request.source.type,
      },
    });
    return delay(response);
  },

  async validatePackaging(request: PackagingValidateRequest) {
    const response: PackagingValidateResponse = {
      ok: true,
      workspacePath: request.workspacePath,
      packageName: request.target.packageName,
      version: request.target.version,
      manifestDigest: 'sha256:1c65eab3e1fb04af11c5ff2e7c54a1f0',
      components: ['docs', 'scripts'],
      warnings: previewWarningDetails.map((item) => item.code),
      warningDetails: previewWarningDetails,
      importReport: previewImportReport,
      errors: [],
    };
    recordAudit(state, {
      action: 'packaging.validate',
      actor: 'web-ui',
      role: 'publisher',
      authMechanism: 'mock',
      authDecision: 'allowed',
      endpoint: '/api/v1/skill-packaging/validate',
      method: 'POST',
      packageName: request.target.packageName,
      version: request.target.version,
      outcome: 'succeeded',
      statusCode: 200,
      details: {
        workspacePath: request.workspacePath,
      },
    });
    return delay(response);
  },

  async packSkill(request: PackagingPackRequest) {
    const response: PackagingPackResponse = {
      workspacePath: request.workspacePath,
      bundlePath: `/tmp/skill-packager/${request.target.packageName.split('/').pop() || 'skill'}-${request.target.version}.tgz`,
      bundleBase64: request.includeBundle ? 'c2tpbGwtYnVuZGxl' : undefined,
      packageName: request.target.packageName,
      version: request.target.version,
      manifestDigest: 'sha256:1c65eab3e1fb04af11c5ff2e7c54a1f0',
      components: ['docs', 'scripts'],
      warnings: previewWarningDetails.map((item) => item.code),
      warningDetails: previewWarningDetails,
      importReport: previewImportReport,
      errors: [],
    };
    recordAudit(state, {
      action: 'packaging.pack',
      actor: 'web-ui',
      role: 'publisher',
      authMechanism: 'mock',
      authDecision: 'allowed',
      endpoint: '/api/v1/skill-packaging/pack',
      method: 'POST',
      packageName: request.target.packageName,
      version: request.target.version,
      outcome: 'succeeded',
      statusCode: 200,
      details: {
        includeBundle: Boolean(request.includeBundle),
      },
    });
    return delay(response);
  },

  async cleanupPackagingWorkspace(request) {
    recordAudit(state, {
      action: 'packaging.cleanup',
      actor: 'web-ui',
      role: 'publisher',
      authMechanism: 'mock',
      authDecision: 'allowed',
      endpoint: '/api/v1/skill-packaging/cleanup',
      method: 'POST',
      packageName: '',
      outcome: 'succeeded',
      statusCode: 200,
      details: {
        workspacePath: request.workspacePath,
      },
    });
    return delay({
      workspacePath: request.workspacePath,
      removed: true,
    });
  },

  async createReleaseRecord(packageName: string, version: string, input: ReleaseRecordInput) {
    const result: ReleaseRecordResult = {
      packageName,
      version,
      created: false,
      idempotentReplay: true,
      versionStatus: 'published',
      installPolicy: 'resolvable',
    };
    const existing = state.versions[packageName]?.find((item) => item.version === version);
    if (!existing) {
      const newVersion: SkillVersionSummary = {
        packageName,
        version,
        versionStatus: 'published',
        installPolicy: 'resolvable',
        publishedAt: nowISOString(),
      };
      state.versions[packageName] = [newVersion, ...(state.versions[packageName] || [])];
      result.created = true;
      result.idempotentReplay = false;
    }
    setLatestSkillState(state, packageName);
    recordAudit(state, {
      action: 'release.record',
      actor: 'web-ui',
      role: 'publisher',
      authMechanism: 'mock',
      authDecision: 'allowed',
      endpoint: `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${version}/release-record`,
      method: 'PUT',
      packageName,
      version,
      outcome: 'succeeded',
      statusCode: result.created ? 201 : 200,
      details: {
        publishRegistry: input.publishRegistry || '',
        created: result.created,
      },
    });
    return delay(result);
  },

  async applySkillGovernanceChange(packageName: string, input: SkillGovernanceChangeInput) {
    const changedAt = nowISOString();
    const record: GovernanceRecord = {
      id: nextGovernanceId(state),
      targetType: 'skill',
      packageName,
      action: input.action,
      reason: input.reason,
      changedBy: 'web-ui',
      changedAt,
    };
    state.governance.unshift(record);

    const skill = state.skills.find((item) => item.packageName === packageName);
    if (skill) {
      skill.status = 'retired';
      skill.updatedAt = changedAt;
    }

    recordAudit(state, {
      action: 'governance.skill',
      actor: 'web-ui',
      role: 'publisher',
      authMechanism: 'mock',
      authDecision: 'allowed',
      endpoint: `/api/v1/skills/${encodeURIComponent(packageName)}/governance-changes`,
      method: 'POST',
      packageName,
      outcome: 'succeeded',
      statusCode: 201,
      details: {
        action: input.action,
      },
    });

    const response: GovernanceChangeResponse = {
      record,
      resultingState: {
        status: 'retired',
      },
    };
    return delay(response);
  },

  async applyVersionGovernanceChange(packageName: string, version: string, input: VersionGovernanceChangeInput) {
    const changedAt = nowISOString();
    const record: GovernanceRecord = {
      id: nextGovernanceId(state),
      targetType: 'version',
      packageName,
      version,
      action: input.action,
      reason: input.reason,
      replacementRecommendation: input.replacementRecommendation,
      changedBy: 'web-ui',
      changedAt,
    };
    state.governance.unshift(record);

    const versionList = state.versions[packageName] || [];
    const target = versionList.find((item) => item.version === version);
    if (target) {
      target.versionStatus = input.action === 'yank' ? 'yanked' : 'deprecated';
      if (input.action === 'yank') {
        target.installPolicy = 'blocked';
      }
    }
    setLatestSkillState(state, packageName);

    recordAudit(state, {
      action: 'governance.version',
      actor: 'web-ui',
      role: 'publisher',
      authMechanism: 'mock',
      authDecision: 'allowed',
      endpoint: `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${version}/governance-changes`,
      method: 'POST',
      packageName,
      version,
      outcome: 'succeeded',
      statusCode: 201,
      details: {
        action: input.action,
      },
    });

    const response: GovernanceChangeResponse = {
      record,
      resultingState: {
        versionStatus: target?.versionStatus || (input.action === 'yank' ? 'yanked' : 'deprecated'),
      },
    };
    return delay(response);
  },
};
