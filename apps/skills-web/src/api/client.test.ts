import { afterEach, describe, expect, it, vi } from 'vitest';
import type { RegistryApiClient, RegistryClientConfig } from './types';
import {
  buildPackagingBundleDownloadPath,
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
    preparePackaging: notUsed,
    validatePackaging: notUsed,
    packSkill: notUsed,
    publishSkill: notUsed,
    cleanupPackagingWorkspace: notUsed,
    createReleaseRecord: notUsed,
    applySkillGovernanceChange: notUsed,
    applyVersionGovernanceChange: notUsed,
    ...overrides,
  } as unknown as RegistryApiClient;
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

  it('fills missing prepare warning details with an empty array', async () => {
    const result = normalizePackagingReview({
      workspacePath: '/tmp/skill-packaging',
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

  it('fills missing bundle metadata with stable defaults', () => {
    expect(normalizeBundleMetadata(null)).toEqual({
      skillManifestDigest: '',
      components: [],
    });
  });

  it('builds the bundle download path through the public facade helper', () => {
    expect(
      buildPackagingBundleDownloadPath('/tmp/skill pack', '@platform.skill/code-review', '1.2.3')
    ).toBe(
      '/api/v1/skill-packaging/bundle?workspacePath=%2Ftmp%2Fskill+pack&packageName=%40platform.skill%2Fcode-review&version=1.2.3'
    );
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
    const preparePackaging = vi.fn().mockResolvedValue({
      workspacePath: '/tmp/mock-workspace',
      packageName: '@platform.skill/code-review',
      version: '1.0.0',
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
        preparePackaging,
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
      clientModule.preparePackaging({
        source: {
          type: 'git',
          location: 'https://github.com/example/skills',
        },
        target: {
          packageName: '@platform.skill/code-review',
          version: '1.0.0',
        },
        metadata: {
          displayName: 'Code Review',
          description: 'Checklist pack',
          sourceType: 'platform_created',
        },
        options: {
          strict: false,
        },
      })
    ).resolves.toEqual({
      workspacePath: '/tmp/mock-workspace',
      packageName: '@platform.skill/code-review',
      version: '1.0.0',
      warnings: ['normalized_missing_capability_content'],
      warningDetails: [],
      errors: [],
    });

    expect(listCuratedSkills).toHaveBeenCalledTimes(1);
    expect(preparePackaging).toHaveBeenCalledTimes(1);
  });

  it('keeps the real-mode facade exports normalized while using the main client seam', async () => {
    const mockConfig: RegistryClientConfig = {
      mode: 'real',
      baseUrl: null,
      role: null,
      actor: null,
      hasToken: true,
    };
    const requestJSON = vi
      .fn()
      .mockResolvedValueOnce({
        items: null,
        pageInfo: null,
      })
      .mockResolvedValueOnce({
        packageName: '@platform.skill/code-review',
        version: '1.0.0',
        versionStatus: 'published',
        installPolicy: 'resolvable',
        publishedAt: '2026-06-02T12:05:00Z',
        registry: {
          npmRegistry: 'https://npm.internal.example.com',
          tarballUrl: 'https://npm.internal.example.com/@platform.skill/code-review/-/code-review-1.0.0.tgz',
          integrity: 'sha512:abc',
          shasum: 'abc',
          publishedAt: '2026-06-02T12:05:00Z',
          distTags: ['latest'],
        },
        bundle: null,
      })
      .mockResolvedValueOnce({
        workspacePath: '/tmp/skill-packaging',
        packageName: '@platform.skill/code-review',
        version: '1.0.0',
        warnings: [],
        errors: null,
      });

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

    const clientModule = await import('./client');

    await expect(clientModule.listSkills()).resolves.toEqual({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    await expect(clientModule.getVersionDetail('@platform.skill/code-review', '1.0.0')).resolves.toEqual({
      packageName: '@platform.skill/code-review',
      version: '1.0.0',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-06-02T12:05:00Z',
      registry: {
        npmRegistry: 'https://npm.internal.example.com',
        tarballUrl: 'https://npm.internal.example.com/@platform.skill/code-review/-/code-review-1.0.0.tgz',
        integrity: 'sha512:abc',
        shasum: 'abc',
        publishedAt: '2026-06-02T12:05:00Z',
        distTags: ['latest'],
      },
      bundle: {
        skillManifestDigest: '',
        components: [],
      },
    });
    await expect(
      clientModule.preparePackaging({
        source: {
          type: 'git',
          location: 'https://github.com/example/skills',
        },
        target: {
          packageName: '@platform.skill/code-review',
          version: '1.0.0',
        },
        metadata: {
          displayName: 'Code Review',
          description: 'Checklist pack',
          sourceType: 'platform_created',
        },
        options: {
          strict: false,
        },
      })
    ).resolves.toEqual({
      workspacePath: '/tmp/skill-packaging',
      packageName: '@platform.skill/code-review',
      version: '1.0.0',
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    expect(requestJSON).toHaveBeenNthCalledWith(1, '/api/v1/skills?pageSize=12');
    expect(requestJSON).toHaveBeenNthCalledWith(
      2,
      '/api/v1/skills/%40platform.skill%2Fcode-review/versions/1.0.0'
    );
    expect(requestJSON).toHaveBeenNthCalledWith(3, '/api/v1/skill-packaging/prepare', {
      method: 'POST',
      body: JSON.stringify({
        source: {
          type: 'git',
          location: 'https://github.com/example/skills',
        },
        target: {
          packageName: '@platform.skill/code-review',
          version: '1.0.0',
        },
        metadata: {
          displayName: 'Code Review',
          description: 'Checklist pack',
          sourceType: 'platform_created',
        },
        options: {
          strict: false,
        },
      }),
    });
  });
});
