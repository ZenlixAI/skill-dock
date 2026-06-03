import type {
  AuditEventRecord,
  CuratedSkillSummary,
  GovernanceChangeResponse,
  GovernanceRecord,
  ImportReport,
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
  RegistryClientConfig,
  ReleaseRecordInput,
  ReleaseRecordResult,
  SkillDetail,
  SkillGovernanceChangeInput,
  SkillSummary,
  SkillVersionDetail,
  SkillVersionSummary,
  VersionGovernanceChangeInput,
  WarningDetail,
} from './types';

const skills: SkillSummary[] = [
  {
    packageName: '@platform.skill/pdf-skill',
    displayName: 'PDF Skill',
    description: 'OCR + extraction + post-processing workflow',
    sourceType: 'imported',
    status: 'active',
    latestVersion: '1.2.0',
    latestVersionStatus: 'published',
    updatedAt: '2026-06-02T20:14:00Z',
  },
  {
    packageName: '@platform.skill/code-review',
    displayName: 'Code Review',
    description: 'Review heuristics and checklist pack',
    sourceType: 'platform_created',
    status: 'active',
    latestVersion: '0.8.4',
    latestVersionStatus: 'published',
    updatedAt: '2026-06-02T12:05:00Z',
  },
  {
    packageName: '@platform.skill/web-scrape',
    displayName: 'Web Scrape',
    description: 'Structured website collection workflow',
    sourceType: 'imported',
    status: 'active',
    latestVersion: '2.0.1',
    latestVersionStatus: 'deprecated',
    updatedAt: '2026-06-01T09:10:00Z',
  },
  {
    packageName: '@platform.skill/legacy-diff',
    displayName: 'Legacy Diff',
    description: 'Historical diffing helper kept for traceability',
    sourceType: 'native',
    status: 'retired',
    latestVersion: '0.5.2',
    latestVersionStatus: 'deprecated',
    updatedAt: '2026-05-28T03:35:00Z',
  },
];

const curatedSkills: CuratedSkillSummary[] = [
  { ...skills[0], curationReason: 'import workflow demo' },
  { ...skills[1], curationReason: 'platform-authored sample' },
  { ...skills[2], curationReason: 'governance timeline sample' },
];

const versions: Record<string, SkillVersionSummary[]> = {
  '@platform.skill/pdf-skill': [
    {
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-06-02T20:14:00Z',
    },
    {
      packageName: '@platform.skill/pdf-skill',
      version: '1.1.1',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-05-27T09:42:00Z',
    },
    {
      packageName: '@platform.skill/pdf-skill',
      version: '1.0.0',
      versionStatus: 'deprecated',
      installPolicy: 'resolvable',
      publishedAt: '2026-05-14T04:18:00Z',
    },
  ],
  '@platform.skill/code-review': [
    {
      packageName: '@platform.skill/code-review',
      version: '0.8.4',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-06-02T12:05:00Z',
    },
  ],
  '@platform.skill/web-scrape': [
    {
      packageName: '@platform.skill/web-scrape',
      version: '2.0.1',
      versionStatus: 'deprecated',
      installPolicy: 'resolvable',
      publishedAt: '2026-06-01T09:10:00Z',
    },
    {
      packageName: '@platform.skill/web-scrape',
      version: '1.9.0',
      versionStatus: 'yanked',
      installPolicy: 'blocked',
      publishedAt: '2026-05-24T08:00:00Z',
    },
  ],
  '@platform.skill/legacy-diff': [
    {
      packageName: '@platform.skill/legacy-diff',
      version: '0.5.2',
      versionStatus: 'deprecated',
      installPolicy: 'blocked',
      publishedAt: '2026-05-28T03:35:00Z',
    },
  ],
};

const governance: GovernanceRecord[] = [
  {
    id: 'gov_01',
    targetType: 'version',
    packageName: '@platform.skill/web-scrape',
    version: '1.9.0',
    action: 'yank',
    reason: 'security issue',
    replacementRecommendation: '2.0.1',
    changedBy: 'svc-publisher',
    changedAt: '2026-06-02T18:56:00Z',
  },
  {
    id: 'gov_02',
    targetType: 'skill',
    packageName: '@platform.skill/legacy-diff',
    action: 'retire',
    reason: 'merged into a newer package set',
    changedBy: 'user_214',
    changedAt: '2026-06-02T15:12:00Z',
  },
  {
    id: 'gov_03',
    targetType: 'version',
    packageName: '@platform.skill/pdf-skill',
    version: '1.0.0',
    action: 'deprecate',
    reason: 'replacement available',
    replacementRecommendation: '1.2.0',
    changedBy: 'svc-publisher',
    changedAt: '2026-06-01T12:20:00Z',
  },
];

const auditEvents: AuditEventRecord[] = [
  {
    id: 'aud_01',
    requestId: 'req_01',
    traceId: 'trace_01',
    action: 'release.record',
    actor: 'svc-publisher',
    role: 'publisher',
    authMechanism: 'header',
    authDecision: 'allowed',
    endpoint: '/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record',
    method: 'PUT',
    packageName: '@platform.skill/pdf-skill',
    version: '1.2.0',
    outcome: 'succeeded',
    statusCode: 201,
    details: {
      created: true,
      idempotentReplay: false,
    },
    createdAt: '2026-06-02T20:15:00Z',
  },
  {
    id: 'aud_02',
    requestId: 'req_02',
    traceId: 'trace_02',
    action: 'packaging.prepare',
    actor: 'svc-publisher',
    role: 'publisher',
    authMechanism: 'header',
    authDecision: 'allowed',
    endpoint: '/api/v1/skill-packaging/prepare',
    method: 'POST',
    packageName: '@platform.skill/pdf-skill',
    version: '1.2.0',
    outcome: 'succeeded',
    statusCode: 200,
    details: {
      sourceType: 'git',
    },
    createdAt: '2026-06-02T20:12:00Z',
  },
  {
    id: 'aud_03',
    requestId: 'req_03',
    traceId: 'trace_03',
    action: 'governance.version',
    actor: 'svc-publisher',
    role: 'publisher',
    authMechanism: 'header',
    authDecision: 'allowed',
    endpoint: '/api/v1/skills/%40platform.skill%2Fweb-scrape/versions/1.9.0/governance-changes',
    method: 'POST',
    packageName: '@platform.skill/web-scrape',
    version: '1.9.0',
    outcome: 'succeeded',
    statusCode: 201,
    details: {
      action: 'yank',
    },
    createdAt: '2026-06-02T18:56:00Z',
  },
];

const previewWarningDetails: WarningDetail[] = [
  {
    code: 'normalized_missing_capability_content',
    severity: 'warning',
    title: 'Generated placeholder capability content',
    message:
      'Imported source lacked commands/, scripts/, docs/ or manifest files, so the packager inserted docs/README.md to keep the bundle publishable.',
    suggestedAction:
      'Replace the placeholder with real capability content before treating this source as the long-term upstream truth.',
  },
  {
    code: 'normalized_rewrote_package_json_identity',
    severity: 'warning',
    title: 'Rewrote package identity',
    message:
      'The upstream package.json declared a different name/version, so the internal target identity was rewritten during prepare.',
    suggestedAction: 'Confirm that the internal package mapping is intentional and record it in the import audit trail.',
  },
];

const previewImportReport: ImportReport = {
  sourceType: 'git',
  strictMode: false,
  normalized: true,
  generatedPackageJson: true,
  generatedCapabilityPlaceholder: true,
  rewrotePackageIdentity: true,
  generatedPaths: ['package.json', 'docs/README.md'],
  warningCodes: ['normalized_missing_capability_content', 'normalized_rewrote_package_json_identity'],
};

function delay<T>(value: T, ms = 120): Promise<T> {
  return new Promise((resolve) => {
    setTimeout(() => resolve(value), ms);
  });
}

function nowISOString() {
  return new Date().toISOString();
}

function nextAuditId() {
  return `aud_${String(auditEvents.length + 1).padStart(2, '0')}`;
}

function nextGovernanceId() {
  return `gov_${String(governance.length + 1).padStart(2, '0')}`;
}

function latestVersionSummary(packageName: string): SkillVersionSummary | undefined {
  return versions[packageName]?.[0];
}

function buildInstallTarget(packageName: string, version?: string): InstallTarget {
  const resolvedVersion = version || latestVersionSummary(packageName)?.version || '1.2.0';
  const foundVersion =
    versions[packageName]?.find((item) => item.version === resolvedVersion) ||
    latestVersionSummary(packageName);
  return {
    packageName,
    version: resolvedVersion,
    npmRegistry: 'https://npm.internal.example.com',
    tarballUrl: `https://npm.internal.example.com/${encodeURIComponent(packageName)}/-/${packageName.split('/').pop()}-${resolvedVersion}.tgz`,
    integrity: 'sha512-0bf7e77d49a2d7...',
    skillManifestDigest: 'sha256:1c65eab3e1fb04af11c5ff2e7c54a1f0',
    versionStatus: foundVersion?.versionStatus || 'published',
  };
}

function buildVersionDetail(packageName: string, version: string): SkillVersionDetail {
  const summary =
    versions[packageName]?.find((item) => item.version === version) ||
    latestVersionSummary(packageName) ||
    versions['@platform.skill/pdf-skill'][0];

  return {
    ...summary,
    registry: {
      npmRegistry: 'https://npm.internal.example.com',
      tarballUrl: `https://npm.internal.example.com/${encodeURIComponent(packageName)}/-/${packageName.split('/').pop()}-${summary.version}.tgz`,
      integrity: 'sha512-0bf7e77d49a2d7...',
      shasum: 'abc123def456',
      publishedAt: summary.publishedAt,
      distTags: summary.versionStatus === 'published' ? ['latest'] : ['legacy'],
    },
    bundle: {
      skillManifestDigest: 'sha256:1c65eab3e1fb04af11c5ff2e7c54a1f0',
      components: ['commands', 'scripts'],
    },
  };
}

function recordAudit(event: Omit<AuditEventRecord, 'id' | 'requestId' | 'traceId' | 'createdAt'>) {
  auditEvents.unshift({
    id: nextAuditId(),
    requestId: `req_${Date.now()}`,
    traceId: `trace_${Date.now()}`,
    createdAt: nowISOString(),
    ...event,
  });
}

function setLatestSkillState(packageName: string) {
  const skill = skills.find((item) => item.packageName === packageName);
  const latest = latestVersionSummary(packageName);
  if (!skill || !latest) {
    return;
  }
  skill.latestVersion = latest.version;
  skill.latestVersionStatus = latest.versionStatus;
  skill.updatedAt = nowISOString();
}

const config: RegistryClientConfig = {
  mode: 'mock',
  baseUrl: null,
  role: null,
  actor: null,
  hasToken: false,
};

export const mockClient: RegistryApiClient = {
  getConfig() {
    return config;
  },

  async listSkills(filters?: ListSkillsFilters) {
    const query = filters?.query?.trim().toLowerCase() || '';
    const filtered = skills.filter((skill) => {
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
    return delay(filtered);
  },

  async listCuratedSkills() {
    return delay(curatedSkills);
  },

  async getSkillDetail(packageName: string) {
    const skill = skills.find((item) => item.packageName === packageName) || skills[0];
    return delay({
      ...skill,
      createdAt: '2026-05-20T02:00:00Z',
    });
  },

  async getSkillVersions(packageName: string) {
    return delay(versions[packageName] || versions['@platform.skill/pdf-skill']);
  },

  async getInstallTarget(packageName: string, version?: string) {
    return delay(buildInstallTarget(packageName, version));
  },

  async getVersionDetail(packageName: string, version: string) {
    return delay(buildVersionDetail(packageName, version));
  },

  async listGovernanceChanges(filters?: ListGovernanceFilters) {
    const filtered = governance.filter((item) => {
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
    return delay(filtered);
  },

  async listAuditEvents(filters?: ListAuditFilters) {
    const filtered = auditEvents.filter((item) => {
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
    return delay(filtered);
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
    recordAudit({
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
    recordAudit({
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
    recordAudit({
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

  async createReleaseRecord(packageName: string, version: string, input: ReleaseRecordInput) {
    const result: ReleaseRecordResult = {
      packageName,
      version,
      created: false,
      idempotentReplay: true,
      versionStatus: 'published',
      installPolicy: 'resolvable',
    };
    const existing = versions[packageName]?.find((item) => item.version === version);
    if (!existing) {
      const newVersion: SkillVersionSummary = {
        packageName,
        version,
        versionStatus: 'published',
        installPolicy: 'resolvable',
        publishedAt: nowISOString(),
      };
      versions[packageName] = [newVersion, ...(versions[packageName] || [])];
      result.created = true;
      result.idempotentReplay = false;
    }
    setLatestSkillState(packageName);
    recordAudit({
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
      id: nextGovernanceId(),
      targetType: 'skill',
      packageName,
      action: input.action,
      reason: input.reason,
      changedBy: 'web-ui',
      changedAt,
    };
    governance.unshift(record);

    const skill = skills.find((item) => item.packageName === packageName);
    if (skill) {
      skill.status = 'retired';
      skill.updatedAt = changedAt;
    }

    recordAudit({
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
      id: nextGovernanceId(),
      targetType: 'version',
      packageName,
      version,
      action: input.action,
      reason: input.reason,
      replacementRecommendation: input.replacementRecommendation,
      changedBy: 'web-ui',
      changedAt,
    };
    governance.unshift(record);

    const versionList = versions[packageName] || [];
    const target = versionList.find((item) => item.version === version);
    if (target) {
      target.versionStatus = input.action === 'yank' ? 'yanked' : 'deprecated';
      if (input.action === 'yank') {
        target.installPolicy = 'blocked';
      }
    }
    setLatestSkillState(packageName);

    recordAudit({
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
