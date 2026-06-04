import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { SkillsPage } from '../api/types';
import { usePaginatedCollection } from './usePaginatedCollection';

type Item = { id: string };
type Filters = { query: string; cursor?: string };

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return { promise, resolve, reject };
}

function page(items: string[], nextCursor?: string | null): SkillsPage<Item> {
  return {
    items: items.map((id) => ({ id })),
    pageInfo: {
      hasNextPage: Boolean(nextCursor),
      nextCursor: nextCursor ?? null,
    },
  };
}

describe('usePaginatedCollection', () => {
  it('invalidates stale pagination controls during a dependency refresh', async () => {
    const alphaFirst = createDeferred<SkillsPage<Item>>();
    const betaFirst = createDeferred<SkillsPage<Item>>();
    const loader = vi.fn((filters: Filters) => {
      if (filters.query === 'alpha') {
        return alphaFirst.promise;
      }
      return betaFirst.promise;
    });

    const { result, rerender } = renderHook(
      ({ query }: { query: string }) =>
        usePaginatedCollection({
          loader,
          filters: { query, cursor: undefined },
          deps: [query],
          preserveStaleItemsOnRefresh: true,
        }),
      {
        initialProps: { query: 'alpha' },
      }
    );

    await act(async () => {
      alphaFirst.resolve(page(['alpha-1'], 'alpha-cursor-2'));
    });

    await waitFor(() => {
      expect(result.current.items.map((item) => item.id)).toEqual(['alpha-1']);
      expect(result.current.pageInfo.nextCursor).toBe('alpha-cursor-2');
    });

    rerender({ query: 'beta' });

    expect(result.current.items.map((item) => item.id)).toEqual(['alpha-1']);
    expect(result.current.refreshing).toBe(true);
    expect(result.current.pageInfo).toEqual({
      hasNextPage: false,
      nextCursor: null,
    });

    act(() => {
      result.current.loadMore();
    });

    expect(loader).toHaveBeenCalledTimes(2);

    await act(async () => {
      betaFirst.resolve(page(['beta-1'], 'beta-cursor-2'));
    });

    await waitFor(() => {
      expect(result.current.items.map((item) => item.id)).toEqual(['beta-1']);
      expect(result.current.pageInfo.nextCursor).toBe('beta-cursor-2');
    });
  });

  it('ignores stale loadMore results after a newer refresh starts', async () => {
    const alphaFirst = createDeferred<SkillsPage<Item>>();
    const alphaSecond = createDeferred<SkillsPage<Item>>();
    const betaFirst = createDeferred<SkillsPage<Item>>();
    const loader = vi.fn((filters: Filters) => {
      if (filters.query === 'alpha' && !filters.cursor) {
        return alphaFirst.promise;
      }
      if (filters.query === 'alpha' && filters.cursor === 'alpha-cursor-2') {
        return alphaSecond.promise;
      }
      return betaFirst.promise;
    });

    const { result, rerender } = renderHook(
      ({ query }: { query: string }) =>
        usePaginatedCollection({
          loader,
          filters: { query, cursor: undefined },
          deps: [query],
          preserveStaleItemsOnRefresh: true,
        }),
      {
        initialProps: { query: 'alpha' },
      }
    );

    await act(async () => {
      alphaFirst.resolve(page(['alpha-1'], 'alpha-cursor-2'));
    });

    await waitFor(() => {
      expect(result.current.items.map((item) => item.id)).toEqual(['alpha-1']);
    });

    act(() => {
      result.current.loadMore();
    });

    expect(loader).toHaveBeenCalledTimes(2);

    rerender({ query: 'beta' });

    await act(async () => {
      betaFirst.resolve(page(['beta-1']));
    });

    await waitFor(() => {
      expect(result.current.items.map((item) => item.id)).toEqual(['beta-1']);
      expect(result.current.pageInfo.hasNextPage).toBe(false);
    });

    await act(async () => {
      alphaSecond.resolve(page(['alpha-2']));
    });

    await waitFor(() => {
      expect(result.current.items.map((item) => item.id)).toEqual(['beta-1']);
    });
  });
});
