import { afterEach, describe, expect, it, vi } from 'vitest';
import type { RegistryApiClient, RegistryClientConfig } from './types';
import {
  normalizeBundleMetadata,
  normalizePackagingReview,
  normalizeSkillsPage,
} from './client';

function createMockApiClient(overrides: Partial<RegistryApiClient>): RegistryApiClient {
  const notUsed = vi.fn();

  return {
    getConfig: notUsed,
    getSession: notUsed,
    loginSession: notUsed,
    logoutSession: notUsed,
    listSkills: notUsed,
    listCuratedSkills: notUsed,
    getSkillDetail: notUsed,
    getSkillVersions: notUsed,
    getInstallTarget: notUsed,
    getVersionDetail: notUsed,
    listGovernanceChanges: notUsed,
    listAuditEvents: notUsed,
    importReviewPackaging: notUsed,
    publishSkill: notUsed,
    createReleaseRecord: notUsed,
    applySkillGovernanceChange: notUsed,
    applyVersionGovernanceChange: notUsed,
    ...overrides,
  } as unknown as RegistryApiClient;
}

async function importWithRequestJSON(requestJSON: ReturnType<typeof vi.fn>) {
  const mockConfig: RegistryClientConfig = {
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: true,
  };

  vi.doMock('./config', () => ({
    registryClientConfig: mockConfig,
  }));
  vi.doMock('./httpClient', () => ({
    requestJSON,
    RegistryHttpError: class RegistryHttpError extends Error {},
  }));
  vi.doMock('./mockClient', () => ({
    mockClient: createMockApiClient({
      getConfig: () => ({
        mode: 'mock',
        baseUrl: null,
        role: null,
        actor: null,
        hasToken: false,
      }),
    }),
  }));

  return import('./client');
}

afterEach(() => {
  vi.resetModules();
  vi.clearAllMocks();
  vi.unmock('./client');
  vi.unmock('./config');
  vi.unmock('./httpClient');
  vi.unmock('./mockClient');
});

describe('apiClient real-response normalization', () => {
  it('normalizes null governance items into an empty array', async () => {
    const result = normalizeSkillsPage({
      items: null,
      pageInfo: {
        nextCursor: null,
        hasNextPage: false,
      },
    });

    expect(result).toEqual({
      items: [],
      pageInfo: {
        nextCursor: null,
        hasNextPage: false,
      },
    });
  });

  it('fills missing review warning details with an empty array', async () => {
    const result = normalizePackagingReview({
      packageName: '@platform.skill/e2e-import-smoke',
      version: '0.0.1',
      warnings: [],
      importReport: {
        sourceType: 'platform_created',
        strictMode: false,
        normalized: false,
        generatedPackageJson: false,
        generatedCapabilityPlaceholder: false,
        rewrotePackageIdentity: false,
      },
      errors: [],
    });

    expect(result.warningDetails).toEqual([]);
    expect(result.errors).toEqual([]);
  });

  it('normalizes import-review responses without workspace fields', async () => {
    const requestJSON = vi.fn().mockResolvedValue({
      ok: true,
      packageName: '@platform.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
      components: ['commands'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    const clientModule = await importWithRequestJSON(requestJSON);
    const response = await clientModule.importReviewPackaging({
      source: { type: 'upload_archive', archiveBase64: 'ZmFrZQ==' },
      target: { packageName: '@platform.skill/zoom-out', version: '1.2.0' },
      metadata: { displayName: 'Zoom Out', description: 'Broader context' },
      options: {},
    });

    expect('workspacePath' in response).toBe(false);
    expect(response.ok).toBe(true);
  });

  it('sends stateless publish requests without workspace fields', async () => {
    const requestJSON = vi.fn().mockResolvedValue({
      packageName: '@platform.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
      components: ['commands'],
      publishedToRegistry: true,
      created: true,
      idempotentReplay: false,
      versionStatus: 'published',
      installPolicy: 'resolvable',
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    const clientModule = await importWithRequestJSON(requestJSON);
    await clientModule.publishSkill({
      source: { type: 'upload_archive', archiveBase64: 'ZmFrZQ==' },
      target: { packageName: '@platform.skill/zoom-out', version: '1.2.0' },
      metadata: { displayName: 'Zoom Out', description: 'Broader context' },
      options: {},
      review: {
        manifestDigest: 'sha256:test',
        components: ['commands'],
      },
    });

    expect(requestJSON).toHaveBeenCalledWith('/api/v1/skill-packaging/publish', {
      method: 'POST',
      body: JSON.stringify({
        source: { type: 'upload_archive', archiveBase64: 'ZmFrZQ==' },
        target: { packageName: '@platform.skill/zoom-out', version: '1.2.0' },
        metadata: { displayName: 'Zoom Out', description: 'Broader context' },
        options: {},
        review: {
          manifestDigest: 'sha256:test',
          components: ['commands'],
        },
      }),
    });
  });

  it('fills missing bundle metadata with stable defaults', () => {
    expect(normalizeBundleMetadata(null)).toEqual({
      skillManifestDigest: '',
      components: [],
    });
  });
});

describe('apiClient facade', () => {
  it('uses the configured mock client through the stable facade exports', async () => {
    const mockConfig: RegistryClientConfig = {
      mode: 'mock',
      baseUrl: null,
      role: null,
      actor: null,
      hasToken: false,
    };
    const listCuratedSkills = vi.fn().mockResolvedValue([
      {
        packageName: '@platform.skill/code-review',
        displayName: 'Code Review',
        description: 'Checklist pack',
        sourceType: 'platform_created',
        status: 'active',
        latestVersion: '1.0.0',
        latestVersionStatus: 'published',
        updatedAt: '2026-06-02T12:05:00Z',
        curationReason: 'sample',
      },
    ]);
    const importReviewPackaging = vi.fn().mockResolvedValue({
      ok: true,
      packageName: '@platform.skill/code-review',
      version: '1.0.0',
      manifestDigest: 'sha256:test',
      components: ['commands'],
      warnings: ['normalized_missing_capability_content'],
      warningDetails: [],
      errors: [],
    });

    vi.doMock('./config', () => ({
      registryClientConfig: mockConfig,
    }));
    vi.doMock('./mockClient', () => ({
      mockClient: createMockApiClient({
        getConfig: () => mockConfig,
        listCuratedSkills,
        importReviewPackaging,
      }),
    }));
    vi.doMock('./httpClient', () => ({
      requestJSON: vi.fn(),
      RegistryHttpError: class RegistryHttpError extends Error {},
    }));

    const clientModule = await import('./client');

    await expect(clientModule.listCuratedSkills()).resolves.toEqual([
      expect.objectContaining({
        packageName: '@platform.skill/code-review',
      }),
    ]);
    await expect(
      clientModule.importReviewPackaging({
        source: {
          type: 'upload_archive',
          archiveBase64: 'YXJjaGl2ZQ==',
        },
        target: {
          packageName: '@platform.skill/code-review',
          version: '1.0.0',
        },
        metadata: {
          displayName: 'Code Review',
          description: 'Checklist pack',
        },
        options: {},
      })
    ).resolves.toEqual(
      expect.objectContaining({
        packageName: '@platform.skill/code-review',
        manifestDigest: 'sha256:test',
      })
    );
    expect(importReviewPackaging).toHaveBeenCalledTimes(1);
  });
});
