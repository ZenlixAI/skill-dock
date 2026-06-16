import type {
  PackagingImportReviewResponse,
  PackagingPublishResponse,
  RegistryRequestError,
} from '../../api/types';
import {
  inferMutationFailureOutcome,
  writeMutationResultSearchParams,
} from '../../components/MutationResultPanel';
import { buildReleaseHandoffPath } from '../../lib/releaseHandoff';

type WorkflowInputs = {
  reviewResult: PackagingImportReviewResponse | null;
  publishResult: PackagingPublishResponse | null;
  publishRegistry: string;
};

type WorkflowReadiness = {
  detail: string;
  tone: 'good' | 'warn';
  label: string;
};

export function derivePackagingViewState({
  reviewResult,
  publishResult,
  publishRegistry,
}: WorkflowInputs) {
  const currentStage = publishResult
    ? 'published'
    : reviewResult
      ? 'reviewed'
        : 'draft';
  const releaseHandoffPath = reviewResult
    ? buildReleaseHandoffPath({
        packageName: reviewResult.packageName,
        version: reviewResult.version,
        publishRegistry: publishRegistry.trim() || undefined,
        manifestDigest: reviewResult.manifestDigest,
        components: reviewResult.components,
      })
    : null;
  const reviewReadiness: WorkflowReadiness = reviewResult
    ? {
        detail: 'manifest digest 和 components 已准备好交接到发布审阅',
        tone: 'good',
        label: '就绪',
      }
    : {
        detail: '等待生成导入审阅结果',
        tone: 'warn',
        label: '等待中',
      };
  const publishReadiness: WorkflowReadiness = publishResult
    ? {
        detail: 'registry publish 和 release record 都已完成',
        tone: 'good',
        label: '已完成',
      }
    : reviewResult?.ok
      ? {
          detail: '审阅结果已确认，可以执行发布',
          tone: 'good',
          label: '可发布',
        }
      : reviewResult
        ? {
            detail: '先处理 errors，再继续发布',
            tone: 'warn',
            label: '待处理',
          }
    : {
        detail: '还没有审阅结果',
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
          detail: 'manifest digest 和 components 已准备好交接到发布审阅',
          tone: 'good',
          label: '已就绪',
        }
      : {
          detail: '导入审阅还没完成',
          tone: 'warn',
          label: '空闲',
        };

  return {
    currentStage,
    currentStageSummary:
      currentStage === 'published'
        ? 'registry publish 和 release record 都已完成。'
        : currentStage === 'reviewed'
          ? '导入审阅已完成，可以继续发布。'
            : '先填写 source 和 target。',
    activeStepIndex: publishResult ? 2 : reviewResult ? 1 : 0,
    canPublish: Boolean(reviewResult?.ok),
    releaseHandoffPath,
    releaseStatus: publishResult ? 'published' : releaseHandoffPath ? 'armed' : 'idle',
    releaseSummary: publishResult
      ? '后端已经完成 registry publish 和 release record。'
      : releaseHandoffPath
      ? '导入审阅结果已就绪，下一步是确认后执行一次性发布。'
      : '完成导入审阅后才会开放下游 handoff。',
    reviewReadiness,
    publishReadiness,
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
