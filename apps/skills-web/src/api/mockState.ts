import type {
  AuditEventRecord,
  CuratedSkillSummary,
  GovernanceRecord,
  ImportReport,
  InstallTarget,
  SkillsPage,
  SkillSummary,
  SkillVersionDetail,
  SkillVersionSummary,
  WarningDetail,
} from './types';

const initialSkills: SkillSummary[] = [
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

export const curatedSkills: CuratedSkillSummary[] = [
  { ...initialSkills[0], curationReason: 'import workflow demo' },
  { ...initialSkills[1], curationReason: 'platform-authored sample' },
  { ...initialSkills[2], curationReason: 'governance timeline sample' },
];

const initialVersions: Record<string, SkillVersionSummary[]> = {
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

const initialGovernance: GovernanceRecord[] = [
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

const initialAuditEvents: AuditEventRecord[] = [
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

export type MockState = {
  skills: SkillSummary[];
  versions: Record<string, SkillVersionSummary[]>;
  governance: GovernanceRecord[];
  auditEvents: AuditEventRecord[];
};

export const previewWarningDetails: WarningDetail[] = [
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
    suggestedAction:
      'Confirm that the internal package mapping is intentional and record it in the import audit trail.',
  },
];

export const previewImportReport: ImportReport = {
  sourceType: 'git',
  strictMode: false,
  normalized: true,
  generatedPackageJson: true,
  generatedCapabilityPlaceholder: true,
  rewrotePackageIdentity: true,
  generatedPaths: ['package.json', 'docs/README.md'],
  warningCodes: [
    'normalized_missing_capability_content',
    'normalized_rewrote_package_json_identity',
  ],
};

export function createMockState(): MockState {
  return {
    skills: initialSkills.map((skill) => ({ ...skill })),
    versions: structuredClone(initialVersions),
    governance: initialGovernance.map((record) => ({ ...record })),
    auditEvents: initialAuditEvents.map((event) => ({ ...event })),
  };
}

export function delay<T>(value: T, ms = 120): Promise<T> {
  return new Promise((resolve) => {
    setTimeout(() => resolve(value), ms);
  });
}

function nowISOString() {
  return new Date().toISOString();
}

function nextAuditId(state: MockState) {
  return `aud_${String(state.auditEvents.length + 1).padStart(2, '0')}`;
}

export function nextGovernanceId(state: MockState) {
  return `gov_${String(state.governance.length + 1).padStart(2, '0')}`;
}

export function latestVersionSummary(
  state: MockState,
  packageName: string
): SkillVersionSummary | undefined {
  return state.versions[packageName]?.[0];
}

export function paginateItems<T>(
  items: T[],
  options?: {
    cursor?: string;
    pageSize?: number;
  }
): SkillsPage<T> {
  const pageSize = options?.pageSize ?? 10;
  const startIndex = options?.cursor ? Number.parseInt(options.cursor, 10) || 0 : 0;
  const pagedItems = items.slice(startIndex, startIndex + pageSize);
  const nextIndex = startIndex + pagedItems.length;

  return {
    items: pagedItems,
    pageInfo: {
      hasNextPage: nextIndex < items.length,
      nextCursor: nextIndex < items.length ? String(nextIndex) : null,
    },
  };
}

export function buildInstallTarget(
  state: MockState,
  packageName: string,
  version?: string
): InstallTarget {
  const resolvedVersion = version || latestVersionSummary(state, packageName)?.version || '1.2.0';
  const foundVersion =
    state.versions[packageName]?.find((item) => item.version === resolvedVersion) ||
    latestVersionSummary(state, packageName);
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

export function buildVersionDetail(
  state: MockState,
  packageName: string,
  version: string
): SkillVersionDetail {
  const summary =
    state.versions[packageName]?.find((item) => item.version === version) ||
    latestVersionSummary(state, packageName) ||
    state.versions['@platform.skill/pdf-skill'][0];

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

export function recordAudit(
  state: MockState,
  event: Omit<AuditEventRecord, 'id' | 'requestId' | 'traceId' | 'createdAt'>
) {
  state.auditEvents.unshift({
    id: nextAuditId(state),
    requestId: `req_${Date.now()}`,
    traceId: `trace_${Date.now()}`,
    createdAt: nowISOString(),
    ...event,
  });
}

export function setLatestSkillState(state: MockState, packageName: string) {
  const skill = state.skills.find((item) => item.packageName === packageName);
  const latest = latestVersionSummary(state, packageName);
  if (!skill || !latest) {
    return;
  }
  skill.latestVersion = latest.version;
  skill.latestVersionStatus = latest.versionStatus;
  skill.updatedAt = nowISOString();
}
