import { useState } from 'react';
import {
  applyVersionGovernanceChange,
  createReleaseRecord,
} from '../../api/client';
import { normalizeThrownRegistryError } from '../../api/httpClient';
import { inferMutationFailureOutcome } from '../../components/MutationResultPanel';
import {
  createMutationState,
  type CatalogMutationState,
} from '../catalog-mutations/useCatalogMutation';
import {
  getGovernanceActionLabel,
  getVersionStatusLabel,
} from '../../lib/installCommand';

export type VersionLogEntry = {
  id: string;
  action: 'release-record' | 'governance';
  outcome: 'succeeded' | 'failed';
  summary: string;
};

type MutationResultPayload = {
  action: string;
  outcome: 'succeeded' | 'failed' | 'forbidden' | 'unauthorized';
  message: string;
  requestId?: string | null;
  packageName?: string | null;
  version?: string | null;
};

type UseVersionDetailActionsOptions = {
  packageName: string;
  version: string;
  onMutationResult: (result: MutationResultPayload) => void;
  onRefresh: () => void;
};

export function useVersionDetailActions(options: UseVersionDetailActionsOptions) {
  const { packageName, version, onMutationResult, onRefresh } = options;
  const [activityLog, setActivityLog] = useState<VersionLogEntry[]>([]);
  const [actionState, setActionState] = useState<CatalogMutationState<'release' | 'governance'>>(
    createMutationState()
  );

  function appendLog(action: VersionLogEntry['action'], outcome: VersionLogEntry['outcome'], summary: string) {
    setActivityLog((current) => [
      {
        id: `${action}-${Date.now()}`,
        action,
        outcome,
        summary,
      },
      ...current,
    ]);
  }

  async function handleReleaseRecord(input: {
    publishRegistry: string;
    manifestDigest: string;
    components: string[];
  }) {
    setActionState({
      ...createMutationState<'release' | 'governance'>(),
      loading: 'release',
    });
    try {
      const response = await createReleaseRecord(packageName, version, {
        publishRegistry: input.publishRegistry.trim() || undefined,
        manifestDigest: input.manifestDigest.trim() || undefined,
        components: input.components.length ? input.components : undefined,
      });
      const successMessage = response.created
        ? '已写入新的发布记录。'
        : '发布记录已存在，后端返回幂等重放结果。';
      setActionState({
        loading: false,
        error: null,
        success: successMessage,
      });
      onMutationResult({
        action: '发布记录',
        outcome: 'succeeded',
        message: successMessage,
        packageName,
        version,
      });
      appendLog(
        'release-record',
        'succeeded',
        response.created ? '已写入新的发布记录。' : '发布记录返回幂等重放结果。'
      );
      onRefresh();
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      appendLog('release-record', 'failed', normalizedError.message);
      setActionState({
        loading: false,
        error: normalizedError.message,
        success: null,
      });
      onMutationResult({
        action: '发布记录',
        outcome,
        message: normalizedError.message,
        requestId: normalizedError.requestId,
        packageName,
        version,
      });
    }
  }

  async function handleVersionGovernance(input: {
    action: 'deprecate' | 'yank';
    reason: string;
    replacementRecommendation: string;
  }) {
    setActionState({
      ...createMutationState<'release' | 'governance'>(),
      loading: 'governance',
    });
    try {
      const response = await applyVersionGovernanceChange(packageName, version, {
        action: input.action,
        reason: input.reason.trim(),
        replacementRecommendation:
          input.replacementRecommendation.trim() === ''
            ? undefined
            : input.replacementRecommendation.trim(),
      });
      const successMessage = `已提交${getGovernanceActionLabel(response.record.action)}。当前版本状态：${
        getVersionStatusLabel(response.resultingState.versionStatus || 'unknown')
      }`;
      setActionState({
        loading: false,
        error: null,
        success: successMessage,
      });
      onMutationResult({
        action: getGovernanceActionLabel(response.record.action),
        outcome: 'succeeded',
        message: successMessage,
        packageName,
        version,
      });
      appendLog('governance', 'succeeded', `已应用${getGovernanceActionLabel(response.record.action)}治理动作。`);
      onRefresh();
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      appendLog('governance', 'failed', normalizedError.message);
      setActionState({
        loading: false,
        error: normalizedError.message,
        success: null,
      });
      onMutationResult({
        action: getGovernanceActionLabel(input.action),
        outcome,
        message: normalizedError.message,
        requestId: normalizedError.requestId,
        packageName,
        version,
      });
    }
  }

  return {
    actionState,
    activityLog,
    handleReleaseRecord,
    handleVersionGovernance,
  };
}
