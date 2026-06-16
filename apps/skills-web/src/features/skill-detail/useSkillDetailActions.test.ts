import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as client from '../../api/client';
import { useSkillDetailActions } from './useSkillDetailActions';

vi.mock('../../api/client', () => ({
  applySkillGovernanceChange: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe('useSkillDetailActions', () => {
  it('retires a skill and records the resulting status in local state', async () => {
    vi.mocked(client.applySkillGovernanceChange).mockResolvedValue({
      record: {
        id: 'chg_123',
        action: 'retire',
        reason: 'merged into successor skill',
        targetType: 'skill',
        packageName: '@platform.skill/agent-notes',
        changedAt: '2026-01-01T00:00:00Z',
        changedBy: 'operator',
      },
      resultingState: {
        status: 'retired',
      },
    });

    const onMutationResult = vi.fn();
    const onRefresh = vi.fn();
    const { result } = renderHook(() =>
      useSkillDetailActions({
        packageName: '@platform.skill/agent-notes',
        onMutationResult,
        onRefresh,
      })
    );

    await act(async () => {
      await result.current.handleRetireSkill('merged into successor skill');
    });

    await waitFor(() =>
      expect(client.applySkillGovernanceChange).toHaveBeenCalledWith('@platform.skill/agent-notes', {
        action: 'retire',
        reason: 'merged into successor skill',
      })
    );
    expect(result.current.submitState.success).toBe('已提交技能退役。当前状态：已退役');
    expect(result.current.activityLog[0]?.summary).toBe('已退役 @platform.skill/agent-notes。');
    expect(onRefresh).toHaveBeenCalledTimes(1);
    expect(onMutationResult).toHaveBeenCalledWith(
      expect.objectContaining({
        action: '退役',
        outcome: 'succeeded',
        packageName: '@platform.skill/agent-notes',
      })
    );
  });
});
