import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as client from '../../api/client';
import { useVersionDetailActions } from './useVersionDetailActions';

vi.mock('../../api/client', () => ({
  applyVersionGovernanceChange: vi.fn(),
  createReleaseRecord: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe('useVersionDetailActions', () => {
  it('applies version governance and records the resulting status in local state', async () => {
    vi.mocked(client.applyVersionGovernanceChange).mockResolvedValue({
      record: {
        id: 'chg_123',
        action: 'yank',
        reason: 'security issue',
        targetType: 'version',
        packageName: '@platform.skill/agent-notes',
        version: '1.0.0',
        changedAt: '2026-01-01T00:00:00Z',
        changedBy: 'operator',
      },
      resultingState: {
        versionStatus: 'yanked',
      },
    });

    const onMutationResult = vi.fn();
    const onRefresh = vi.fn();
    const { result } = renderHook(() =>
      useVersionDetailActions({
        packageName: '@platform.skill/agent-notes',
        version: '1.0.0',
        onMutationResult,
        onRefresh,
      })
    );

    await act(async () => {
      await result.current.handleVersionGovernance({
        action: 'yank',
        reason: 'security issue',
        replacementRecommendation: '0.9.9',
      });
    });

    await waitFor(() =>
      expect(client.applyVersionGovernanceChange).toHaveBeenCalledWith(
        '@platform.skill/agent-notes',
        '1.0.0',
        {
          action: 'yank',
          reason: 'security issue',
          replacementRecommendation: '0.9.9',
        }
      )
    );
    expect(result.current.actionState.success).toBe('已提交撤回。当前版本状态：已撤回');
    expect(result.current.activityLog[0]?.summary).toBe('已应用撤回治理动作。');
    expect(onRefresh).toHaveBeenCalledTimes(1);
    expect(onMutationResult).toHaveBeenCalledWith(
      expect.objectContaining({
        action: '撤回',
        outcome: 'succeeded',
        packageName: '@platform.skill/agent-notes',
        version: '1.0.0',
      })
    );
  });
});
