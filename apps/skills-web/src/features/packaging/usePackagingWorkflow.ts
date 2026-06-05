import type {
  PackagingPackResponse,
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
  publishRegistry,
}: WorkflowInputs) {
  const currentStage = packResult
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
        detail: '等待工作目录',
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
  const releaseReadiness: WorkflowReadiness = releaseHandoffPath
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
      currentStage === 'pack_complete'
        ? 'bundle 已可交接到 release review。'
        : currentStage === 'validated'
          ? '校验完成，等待打包。'
          : currentStage === 'prepared'
            ? 'workspace 已准备，建议继续 validate。'
            : '先填写 source 和 target。',
    activeStepIndex: packResult ? 3 : validateResult ? 2 : prepareResult ? 1 : 0,
    canValidate: Boolean(prepareResult?.workspacePath),
    canPack: Boolean(validateResult?.ok),
    canCleanup: Boolean(prepareResult?.workspacePath),
    releaseHandoffPath,
    workspaceStatus: prepareResult?.workspacePath ? 'ready' : 'pending',
    releaseStatus: releaseHandoffPath ? 'armed' : 'idle',
    releaseSummary: releaseHandoffPath
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
