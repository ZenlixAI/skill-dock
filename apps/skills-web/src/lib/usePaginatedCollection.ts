import { startTransition, useEffect, useState } from 'react';
import type { PageInfo, SkillsPage } from '../api/types';

const emptyPageInfo: PageInfo = {
  hasNextPage: false,
  nextCursor: null,
};

interface PaginatedState<T> {
  items: T[];
  pageInfo: PageInfo;
  loading: boolean;
  loadingMore: boolean;
  error: string | null;
}

interface UsePaginatedCollectionOptions<T, F extends { cursor?: string }> {
  loader: (filters: F) => Promise<SkillsPage<T>>;
  filters: F;
  deps: readonly unknown[];
}

export function usePaginatedCollection<T, F extends { cursor?: string }>({
  loader,
  filters,
  deps,
}: UsePaginatedCollectionOptions<T, F>): PaginatedState<T> & { loadMore: () => void } {
  const [state, setState] = useState<PaginatedState<T>>({
    items: [],
    pageInfo: emptyPageInfo,
    loading: true,
    loadingMore: false,
    error: null,
  });

  useEffect(() => {
    let cancelled = false;

    setState({
      items: [],
      pageInfo: emptyPageInfo,
      loading: true,
      loadingMore: false,
      error: null,
    });

    loader({ ...filters, cursor: undefined })
      .then((page) => {
        if (cancelled) {
          return;
        }
        startTransition(() => {
          setState({
            items: page.items,
            pageInfo: page.pageInfo,
            loading: false,
            loadingMore: false,
            error: null,
          });
        });
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return;
        }
        const message = error instanceof Error ? error.message : String(error);
        startTransition(() => {
          setState({
            items: [],
            pageInfo: emptyPageInfo,
            loading: false,
            loadingMore: false,
            error: message,
          });
        });
      });

    return () => {
      cancelled = true;
    };
  }, deps);

  function loadMore() {
    const nextCursor = state.pageInfo.nextCursor;
    if (state.loading || state.loadingMore || !state.pageInfo.hasNextPage || !nextCursor) {
      return;
    }

    setState((current) => ({
      ...current,
      loadingMore: true,
      error: null,
    }));

    loader({ ...filters, cursor: nextCursor })
      .then((page) => {
        startTransition(() => {
          setState((current) => ({
            items: [...current.items, ...page.items],
            pageInfo: page.pageInfo,
            loading: false,
            loadingMore: false,
            error: null,
          }));
        });
      })
      .catch((error: unknown) => {
        const message = error instanceof Error ? error.message : String(error);
        startTransition(() => {
          setState((current) => ({
            ...current,
            loadingMore: false,
            error: message,
          }));
        });
      });
  }

  return {
    ...state,
    loadMore,
  };
}
