import { describe, expect, it } from 'vitest';
import { RegistryHttpError, normalizeThrownRegistryError } from '../../api/httpClient';
import { inferMutationFailureOutcome } from '../../components/MutationResultPanel';
import { buildReleaseHandoffPath } from '../../lib/releaseHandoff';
import { derivePackagingViewState, toMutationResult } from './packagingWorkflowState';

describe('packaging workflow view state', () => {
  it('moves to reviewed after a successful import-review result', () => {
    expect(
      derivePackagingViewState({
        reviewResult: {
          ok: true,
          packageName: '@company.skill/zoom-out',
          version: '1.2.0',
          manifestDigest: 'sha256:abc',
          components: ['docs'],
          warnings: [],
          warningDetails: [],
          errors: [],
        },
        publishResult: null,
        publishRegistry: 'https://npm.internal.example.com',
      }).currentStage
    ).toBe('reviewed');
  });

  it('builds a handoff path once import-review completes', () => {
    const state = derivePackagingViewState({
      reviewResult: {
        ok: true,
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
        warnings: [],
        warningDetails: [],
        errors: [],
      },
      publishResult: null,
      publishRegistry: 'https://npm.internal.example.com',
    });

    expect(state.releaseHandoffPath).toBe(
      buildReleaseHandoffPath({
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        publishRegistry: 'https://npm.internal.example.com',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
      })
    );
    expect(state.currentStageSummary).toBe('导入审阅已完成，可以继续发布。');
    expect(state.releaseReadiness).toEqual({
      detail: 'manifest digest 和 components 已准备好交接到发布审阅',
      tone: 'good',
      label: '已就绪',
    });
  });

  it('describes the draft state without workspace handles', () => {
    const state = derivePackagingViewState({
      reviewResult: null,
      publishResult: null,
      publishRegistry: 'https://npm.internal.example.com',
    });

    expect(state.reviewReadiness).toEqual({
      detail: '等待生成导入审阅结果',
      tone: 'warn',
      label: '等待中',
    });
  });

  it('shows the published stage once backend publish succeeds', () => {
    const state = derivePackagingViewState({
      reviewResult: {
        ok: true,
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
        warnings: [],
        warningDetails: [],
        errors: [],
      },
      publishResult: {
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
        publishedToRegistry: true,
        created: true,
        idempotentReplay: false,
        versionStatus: 'published',
        installPolicy: 'resolvable',
        warnings: [],
        warningDetails: [],
        errors: [],
      },
      publishRegistry: 'https://npm.internal.example.com',
    });

    expect(state.currentStage).toBe('published');
    expect(state.currentStageSummary).toBe('registry publish 和 release record 都已完成。');
    expect(state.releaseReadiness).toEqual({
      detail: 'registry publish 和 release record 都已完成',
      tone: 'good',
      label: '已完成',
    });
  });

  it('maps a thrown 403 into a forbidden mutation result', () => {
    const thrown = new RegistryHttpError('forbidden', 403, {
      error: {
        requestId: 'req_123',
      },
    });
    const result = toMutationResult(
      'publish',
      normalizeThrownRegistryError(thrown),
      inferMutationFailureOutcome(thrown),
      { packageName: '@company.skill/zoom-out', version: '1.2.0', actor: 'tester' }
    );

    expect(result).toEqual({
      action: 'publish',
      outcome: 'forbidden',
      message: 'Current session is not allowed to perform this action.',
      requestId: 'req_123',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      actor: 'tester',
    });
  });
});
