import type {
  PackagingPackResponse,
  PackagingPublishResponse,
  PackagingPrepareResponse,
  PackagingValidateResponse,
  RegistryRequestError,
} from '../../api/types';
import {
  inferMutationFailureOutcome,
  writeMutationResultSearchParams,
} from '../../components/MutationResultPanel';
import { buildReleaseHandoffPath } from '../../lib/releaseHandoff';

type WorkflowInputs = {
  prepareResult: PackagingPrepareResponse | null;
  validateResult: PackagingValidateResponse | null;
  packResult: PackagingPackResponse | null;
  publishResult: PackagingPublishResponse | null;
  publishRegistry: string;
};

type WorkflowReadiness = {
  detail: string;
  tone: 'good' | 'warn';
  label: string;
};

export function derivePackagingViewState({
  prepareResult,
  validateResult,
  packResult,
  publishResult,
  publishRegistry,
}: WorkflowInputs) {
  const currentStage = publishResult
    ? 'published'
    : packResult
      ? 'pack_complete'
    : validateResult
      ? 'validated'
      : prepareResult
        ? 'prepared'
        : 'draft';
  const releaseHandoffPath = packResult
    ? buildReleaseHandoffPath({
        packageName: packResult.packageName,
        version: packResult.version,
        publishRegistry: publishRegistry.trim() || undefined,
        manifestDigest: packResult.manifestDigest,
        components: packResult.components,
        workspacePath: packResult.workspacePath,
        bundlePath: packResult.bundlePath,
      })
    : null;
  const prepareReadiness: WorkflowReadiness = prepareResult?.workspacePath
    ? {
        detail: prepareResult.workspacePath,
        tone: 'good',
        label: '就绪',
      }
    : {
        detail: '等待 workspace handle',
        tone: 'warn',
        label: '等待中',
      };
  const validateReadiness: WorkflowReadiness = validateResult?.ok
    ? {
        detail: '可以进入 pack',
        tone: 'good',
        label: '已通过',
      }
    : {
        detail: '必须先通过再 pack',
        tone: 'warn',
        label: '等待中',
      };
  const releaseReadiness: WorkflowReadiness = publishResult
    ? {
        detail: 'registry publish 和 release record 都已完成',
        tone: 'good',
        label: '已完成',
      }
    : releaseHandoffPath
      ? {
          detail: 'bundle 下载与 release handoff 已就绪',
          tone: 'good',
          label: '已就绪',
        }
      : {
          detail: 'pack 还没完成',
          tone: 'warn',
          label: '空闲',
        };

  return {
    currentStage,
    currentStageSummary:
      currentStage === 'published'
        ? 'registry publish 和 release record 都已完成。'
        : currentStage === 'pack_complete'
        ? 'bundle 已可交接到 release review。'
        : currentStage === 'validated'
          ? '校验完成，等待打包。'
          : currentStage === 'prepared'
            ? 'workspace handle 已准备，建议继续 validate。'
            : '先填写 source 和 target。',
    activeStepIndex: publishResult || packResult ? 3 : validateResult ? 2 : prepareResult ? 1 : 0,
    canValidate: Boolean(prepareResult?.workspacePath),
    canPack: Boolean(validateResult?.ok),
    canPublish: Boolean(validateResult?.ok),
    canCleanup: Boolean(prepareResult?.workspacePath),
    releaseHandoffPath,
    workspaceStatus: prepareResult?.workspacePath ? 'ready' : 'pending',
    releaseStatus: publishResult ? 'published' : releaseHandoffPath ? 'armed' : 'idle',
    releaseSummary: publishResult
      ? '后端已经完成 registry publish 和 release record。'
      : releaseHandoffPath
      ? 'pack 结果已就绪，下一步是下载 bundle、发布到 registry，再写 release record。'
      : '完成 pack 后才会开放下游 handoff。',
    prepareReadiness,
    validateReadiness,
    releaseReadiness,
  };
}

export function toMutationResult(
  action: string,
  error: RegistryRequestError,
  outcome: ReturnType<typeof inferMutationFailureOutcome>,
  defaults: {
    packageName?: string | null;
    version?: string | null;
    actor?: string | null;
    message?: string;
  }
) {
  return {
    action,
    outcome,
    message: defaults.message ?? error.message,
    requestId: error.requestId,
    packageName: defaults.packageName ?? null,
    version: defaults.version ?? null,
    actor: defaults.actor ?? null,
  };
}

export function persistPackagingMutationResult(
  searchParams: URLSearchParams,
  result: Parameters<typeof writeMutationResultSearchParams>[1]
) {
  return writeMutationResultSearchParams(searchParams, result);
}
