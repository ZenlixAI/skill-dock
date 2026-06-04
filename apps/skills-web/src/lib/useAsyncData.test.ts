import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useAsyncData } from './useAsyncData';

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return { promise, resolve, reject };
}

describe('useAsyncData', () => {
  it('clears old data by default when dependencies change', async () => {
    const first = createDeferred<string>();
    const second = createDeferred<string>();
    const loader = vi.fn((resourceId: string) => {
      return resourceId === 'alpha' ? first.promise : second.promise;
    });

    const { result, rerender } = renderHook(
      ({ resourceId }: { resourceId: string }) => useAsyncData(() => loader(resourceId), [resourceId]),
      {
        initialProps: { resourceId: 'alpha' },
      }
    );

    await act(async () => {
      first.resolve('alpha-data');
    });

    await waitFor(() => {
      expect(result.current.data).toBe('alpha-data');
      expect(result.current.loading).toBe(false);
    });

    rerender({ resourceId: 'beta' });

    expect(result.current.data).toBeNull();
    expect(result.current.loading).toBe(true);
    expect(result.current.refreshing).toBe(false);

    await act(async () => {
      second.resolve('beta-data');
    });

    await waitFor(() => {
      expect(result.current.data).toBe('beta-data');
      expect(result.current.loading).toBe(false);
    });
  });

  it('preserves stale data only when explicitly requested', async () => {
    const first = createDeferred<string>();
    const second = createDeferred<string>();
    const loader = vi.fn((resourceId: string) => {
      return resourceId === 'alpha' ? first.promise : second.promise;
    });

    const { result, rerender } = renderHook(
      ({ resourceId }: { resourceId: string }) =>
        useAsyncData(() => loader(resourceId), [resourceId], { preserveStaleData: true }),
      {
        initialProps: { resourceId: 'alpha' },
      }
    );

    await act(async () => {
      first.resolve('alpha-data');
    });

    await waitFor(() => {
      expect(result.current.data).toBe('alpha-data');
      expect(result.current.loading).toBe(false);
    });

    rerender({ resourceId: 'beta' });

    expect(result.current.data).toBe('alpha-data');
    expect(result.current.loading).toBe(false);
    expect(result.current.refreshing).toBe(true);

    await act(async () => {
      second.reject(new Error('Failed to fetch'));
    });

    await waitFor(() => {
      expect(result.current.data).toBe('alpha-data');
      expect(result.current.refreshing).toBe(false);
      expect(result.current.errorDetail?.kind).toBe('network');
    });
  });
});
