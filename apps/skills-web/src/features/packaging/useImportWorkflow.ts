import { useEffect, useRef, useState } from 'react';
import { importReviewPackaging, publishSkill } from '../../api/client';
import { normalizeThrownRegistryError } from '../../api/httpClient';
import type {
  PackagingImportReviewRequest,
  PackagingImportReviewResponse,
  PackagingPublishResponse,
  PackagingSourceType,
  RegistryRequestError,
} from '../../api/types';
import { inferMutationFailureOutcome } from '../../components/MutationResultPanel';
import { toMutationResult } from './packagingWorkflowState';

export type ImportLogEntry = {
  id: string;
  step: 'review' | 'publish';
  outcome: 'succeeded' | 'failed';
  summary: string;
};

export type ImportOperationState = {
  loading: false | 'review' | 'publish';
  error: string | null;
  errorDetail: RegistryRequestError | null;
  success: string | null;
};

type MutationResultPayload = {
  action: string;
  outcome: 'succeeded' | 'failed' | 'forbidden' | 'unauthorized';
  message: string;
  requestId?: string | null;
  packageName?: string | null;
  version?: string | null;
};

type UseImportWorkflowOptions = {
  sourceType: PackagingSourceType;
  authoringError: string | null;
  normalizedPackageName: string;
  version: string;
  buildReviewRequest: () => Promise<PackagingImportReviewRequest>;
  onMutationResult: (result: MutationResultPayload) => void;
  resetKeys: readonly unknown[];
};

function createLocalRequestError(message: string): RegistryRequestError {
  return {
    kind: 'unknown',
    message,
  };
}

function describeImportOperationError(error: RegistryRequestError) {
  if (error.kind === 'auth') {
    return `${error.message} 这次导入操作需要具备 packaging 权限的会话后再重试。`;
  }

  if (error.kind === 'network') {
    return `${error.message} 当前无法连接 packaging 后端，等真实后端恢复后再重试。`;
  }

  return error.message;
}

export function useImportWorkflow(options: UseImportWorkflowOptions) {
  const {
    sourceType,
    authoringError,
    normalizedPackageName,
    version,
    buildReviewRequest,
    onMutationResult,
    resetKeys,
  } = options;
  const [operationState, setOperationState] = useState<ImportOperationState>({
    loading: false,
    error: null,
    errorDetail: null,
    success: null,
  });
  const [reviewResult, setReviewResult] = useState<PackagingImportReviewResponse | null>(null);
  const [publishResult, setPublishResult] = useState<PackagingPublishResponse | null>(null);
  const [activityLog, setActivityLog] = useState<ImportLogEntry[]>([]);
  const inputRevisionRef = useRef(0);

  useEffect(() => {
    inputRevisionRef.current += 1;
    setReviewResult(null);
    setPublishResult(null);
    setOperationState((current) =>
      current.loading === false && current.success === null
        ? current
        : {
            loading: false,
            error: null,
            errorDetail: null,
            success: null,
          }
    );
  }, resetKeys);

  function appendLog(step: ImportLogEntry['step'], outcome: ImportLogEntry['outcome'], summary: string) {
    setActivityLog((current) => [
      {
        id: `${step}-${Date.now()}`,
        step,
        outcome,
        summary,
      },
      ...current,
    ]);
  }

  async function handlePreview() {
    if (sourceType === 'platform_authoring' && authoringError) {
      setOperationState({
        loading: false,
        error: authoringError,
        errorDetail: createLocalRequestError(authoringError),
        success: null,
      });
      return;
    }

    const inputRevision = inputRevisionRef.current;
    setOperationState({
      loading: 'review',
      error: null,
      errorDetail: null,
      success: null,
    });

    try {
      const response = await importReviewPackaging(await buildReviewRequest());
      if (inputRevision !== inputRevisionRef.current) {
        return;
      }
      setReviewResult(response);
      setPublishResult(null);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: '发布预览已生成，可以继续执行一次性发布。',
      });
      onMutationResult({
        action: 'import-review',
        outcome: 'succeeded',
        message: '发布预览已生成，可以继续执行一次性发布。',
        packageName: response.packageName,
        version: response.version,
      });
      appendLog('review', 'succeeded', `已生成 ${response.packageName}@${response.version} 的发布预览。`);
    } catch (error: unknown) {
      if (inputRevision !== inputRevisionRef.current) {
        return;
      }
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      const message = describeImportOperationError(normalizedError);
      appendLog('review', 'failed', normalizedError.message);
      setOperationState({
        loading: false,
        error: normalizedError.message,
        errorDetail: normalizedError,
        success: null,
      });
      onMutationResult(
        toMutationResult('import-review', normalizedError, outcome, {
          packageName: normalizedPackageName || null,
          version: version.trim() || null,
          message,
        })
      );
    }
  }

  async function handlePublish() {
    if (!reviewResult?.ok) {
      const message = '请先执行“检查并生成发布预览”。';
      setOperationState({
        loading: false,
        error: message,
        errorDetail: createLocalRequestError(message),
        success: null,
      });
      return;
    }

    const inputRevision = inputRevisionRef.current;
    setOperationState({
      loading: 'publish',
      error: null,
      errorDetail: null,
      success: null,
    });

    try {
      const response = await publishSkill({
        ...(await buildReviewRequest()),
        review: {
          manifestDigest: reviewResult.manifestDigest,
          components: reviewResult.components,
        },
      });
      if (inputRevision !== inputRevisionRef.current) {
        return;
      }
      setPublishResult(response);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: response.created ? 'skill 已完成一次性发布，并写入 release record。' : '后端返回幂等重放结果。',
      });
      onMutationResult({
        action: 'publish',
        outcome: 'succeeded',
        message: response.created ? 'skill 已完成一次性发布，并写入 release record。' : '后端返回幂等重放结果。',
        packageName: response.packageName,
        version: response.version,
      });
      appendLog(
        'publish',
        'succeeded',
        response.created ? '已完成一次性发布。' : '返回幂等重放发布结果。'
      );
    } catch (error: unknown) {
      if (inputRevision !== inputRevisionRef.current) {
        return;
      }
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      appendLog('publish', 'failed', normalizedError.message);
      setOperationState({
        loading: false,
        error: normalizedError.message,
        errorDetail: normalizedError,
        success: null,
      });
      onMutationResult(
        toMutationResult('publish', normalizedError, outcome, {
          packageName: reviewResult.packageName,
          version: reviewResult.version,
        })
      );
    }
  }

  return {
    operationState,
    reviewResult,
    publishResult,
    activityLog,
    handlePreview,
    handlePublish,
  };
}
