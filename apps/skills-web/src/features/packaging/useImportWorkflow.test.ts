import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as client from '../../api/client';
import { useImportWorkflow } from './useImportWorkflow';

vi.mock('../../api/client', () => ({
  importReviewPackaging: vi.fn(),
  publishSkill: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe('useImportWorkflow', () => {
  it('publishes with review summary from the latest successful import review', async () => {
    const buildReviewRequest = vi.fn().mockResolvedValue({
      source: {
        type: 'platform_authoring',
        files: {
          'workflows/main.md': 'step 1\nstep 2\n',
        },
      },
      target: {
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
      },
      metadata: {
        displayName: 'Zoom Out',
        description: 'Broader context',
        sourceType: 'imported',
      },
      options: {
        publishRegistry: 'https://npm.internal.example.com',
        strict: false,
      },
      includeBundleBase64: false,
    });
    vi.mocked(client.importReviewPackaging).mockResolvedValue({
      ok: true,
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
      components: ['workflows'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.publishSkill).mockResolvedValue({
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
      components: ['workflows'],
      publishedToRegistry: true,
      created: true,
      idempotentReplay: false,
      versionStatus: 'published',
      installPolicy: 'resolvable',
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    const onMutationResult = vi.fn();
    const { result } = renderHook(() =>
      useImportWorkflow({
        sourceType: 'platform_authoring',
        authoringError: null,
        normalizedPackageName: '@company.skill/zoom-out',
        version: '1.2.0',
        buildReviewRequest,
        onMutationResult,
        resetKeys: [],
      })
    );

    await act(async () => {
      await result.current.handlePreview();
    });

    await waitFor(() => expect(result.current.reviewResult?.manifestDigest).toBe('sha256:test'));

    await act(async () => {
      await result.current.handlePublish();
    });

    await waitFor(() =>
      expect(client.publishSkill).toHaveBeenCalledWith(
        expect.objectContaining({
          review: {
            manifestDigest: 'sha256:test',
            components: ['workflows'],
          },
        })
      )
    );
    expect(vi.mocked(client.publishSkill).mock.calls[0]?.[0]).not.toHaveProperty(
      'includeBundleBase64'
    );
    expect(result.current.publishResult?.created).toBe(true);
    expect(result.current.operationState.success).toBe('skill 已完成一次性发布，并写入 release record。');
    expect(onMutationResult).toHaveBeenCalledWith(
      expect.objectContaining({
        action: 'publish',
        outcome: 'succeeded',
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
      })
    );
  });
});
