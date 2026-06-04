import { describe, expect, it } from 'vitest';
import { normalizePackagingReview, normalizeSkillsPage } from './client';

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
});
