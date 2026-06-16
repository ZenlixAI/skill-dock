import { useState } from 'react';
import { applySkillGovernanceChange } from '../../api/client';
import { normalizeThrownRegistryError } from '../../api/httpClient';
import { inferMutationFailureOutcome } from '../../components/MutationResultPanel';
import {
  createMutationState,
  type CatalogMutationState,
} from '../catalog-mutations/useCatalogMutation';
import {
  getGovernanceActionLabel,
  getSkillStatusLabel,
} from '../../lib/installCommand';

export type SkillLogEntry = {
  id: string;
  action: 'retire';
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

type UseSkillDetailActionsOptions = {
  packageName: string;
  onMutationResult: (result: MutationResultPayload) => void;
  onRefresh: () => void;
};

export function useSkillDetailActions(options: UseSkillDetailActionsOptions) {
  const { packageName, onMutationResult, onRefresh } = options;
  const [activityLog, setActivityLog] = useState<SkillLogEntry[]>([]);
  const [submitState, setSubmitState] = useState<CatalogMutationState>(createMutationState());

  function appendLog(outcome: SkillLogEntry['outcome'], summary: string) {
    setActivityLog((current) => [
      {
        id: `retire-${Date.now()}`,
        action: 'retire',
        outcome,
        summary,
      },
      ...current,
    ]);
  }

  async function handleRetireSkill(reason: string) {
    setSubmitState({
      ...createMutationState(),
      loading: true,
    });
    try {
      const response = await applySkillGovernanceChange(packageName, {
        action: 'retire',
        reason: reason.trim(),
      });
      const successMessage = `已提交技能退役。当前状态：${getSkillStatusLabel(response.resultingState.status || 'retired')}`;
      setSubmitState({
        loading: false,
        error: null,
        success: successMessage,
      });
      onMutationResult({
        action: getGovernanceActionLabel('retire'),
        outcome: 'succeeded',
        message: successMessage,
        packageName,
        version: null,
      });
      appendLog('succeeded', `已退役 ${packageName}。`);
      onRefresh();
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      appendLog('failed', normalizedError.message);
      setSubmitState({
        loading: false,
        error: normalizedError.message,
        success: null,
      });
      onMutationResult({
        action: getGovernanceActionLabel('retire'),
        outcome,
        message: normalizedError.message,
        requestId: normalizedError.requestId,
        packageName,
        version: null,
      });
    }
  }

  return {
    submitState,
    activityLog,
    handleRetireSkill,
  };
}
