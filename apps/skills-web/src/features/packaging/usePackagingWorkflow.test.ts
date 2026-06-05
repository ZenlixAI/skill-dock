import { describe, expect, it } from 'vitest';
import { RegistryHttpError, normalizeThrownRegistryError } from '../../api/httpClient';
import { inferMutationFailureOutcome } from '../../components/MutationResultPanel';
import { buildReleaseHandoffPath } from '../../lib/releaseHandoff';
import { derivePackagingViewState, toMutationResult } from './usePackagingWorkflow';

describe('packaging workflow view state', () => {
  it('moves to validated after a successful validate result', () => {
    expect(
      derivePackagingViewState({
        prepareResult: {
          workspacePath: '/tmp/workspace',
          packageName: '@company.skill/zoom-out',
          version: '1.2.0',
          warnings: [],
          warningDetails: [],
          errors: [],
        },
        validateResult: {
          ok: true,
          workspacePath: '/tmp/workspace',
          packageName: '@company.skill/zoom-out',
          version: '1.2.0',
          manifestDigest: 'sha256:abc',
          components: ['docs'],
          warnings: [],
          warningDetails: [],
          errors: [],
        },
        packResult: null,
        publishRegistry: 'https://npm.internal.example.com',
      }).currentStage
    ).toBe('validated');
  });

  it('builds a handoff path once pack completes', () => {
    const state = derivePackagingViewState({
      prepareResult: null,
      validateResult: null,
      packResult: {
        workspacePath: '/tmp/workspace',
        bundlePath: '/tmp/workspace/bundle.tgz',
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
        warnings: [],
        warningDetails: [],
        errors: [],
      },
      publishRegistry: 'https://npm.internal.example.com',
    });

    expect(state.releaseHandoffPath).toBe(
      buildReleaseHandoffPath({
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        publishRegistry: 'https://npm.internal.example.com',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
        workspacePath: '/tmp/workspace',
        bundlePath: '/tmp/workspace/bundle.tgz',
      })
    );
    expect(state.currentStageSummary).toBe('bundle 已可交接到 release review。');
    expect(state.releaseReadiness).toEqual({
      detail: 'bundle 下载与 release handoff 已就绪',
      tone: 'good',
      label: '已就绪',
    });
  });

  it('derives current-stage and readiness copy for the validated state', () => {
    const state = derivePackagingViewState({
      prepareResult: {
        workspacePath: '/tmp/workspace',
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        warnings: [],
        warningDetails: [],
        errors: [],
      },
      validateResult: {
        ok: true,
        workspacePath: '/tmp/workspace',
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        manifestDigest: 'sha256:abc',
        components: ['docs'],
        warnings: [],
        warningDetails: [],
        errors: [],
      },
      packResult: null,
      publishRegistry: 'https://npm.internal.example.com',
    });

    expect(state.currentStageSummary).toBe('校验完成，等待打包。');
    expect(state.prepareReadiness).toEqual({
      detail: '/tmp/workspace',
      tone: 'good',
      label: '就绪',
    });
    expect(state.validateReadiness).toEqual({
      detail: '可以进入 pack',
      tone: 'good',
      label: '已通过',
    });
  });

  it('maps a thrown 403 into a forbidden mutation result', () => {
    const thrown = new RegistryHttpError('forbidden', 403, {
      error: {
        requestId: 'req_123',
      },
    });
    const result = toMutationResult(
      'pack',
      normalizeThrownRegistryError(thrown),
      inferMutationFailureOutcome(thrown),
      { packageName: '@company.skill/zoom-out', version: '1.2.0', actor: 'tester' }
    );

    expect(result).toEqual({
      action: 'pack',
      outcome: 'forbidden',
      message: 'Current session is not allowed to perform this action.',
      requestId: 'req_123',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      actor: 'tester',
    });
  });

  it('prefers an override message when persisting a mutation result', () => {
    const thrown = new RegistryHttpError('forbidden', 403, {
      error: {
        requestId: 'req_123',
      },
    });
    const result = toMutationResult(
      'prepare',
      normalizeThrownRegistryError(thrown),
      inferMutationFailureOutcome(thrown),
      {
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
        actor: 'tester',
        message: 'Current session is not allowed to perform this action. 这次导入操作需要具备 packaging 权限的会话后再重试。',
      }
    );

    expect(result.message).toBe(
      'Current session is not allowed to perform this action. 这次导入操作需要具备 packaging 权限的会话后再重试。'
    );
  });
});
