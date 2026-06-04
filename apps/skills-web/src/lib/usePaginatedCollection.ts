import { startTransition, useEffect, useRef, useState } from 'react';
import { normalizeThrownRegistryError } from '../api/httpClient';
import type { PageInfo, RegistryRequestError, SkillsPage } from '../api/types';

const emptyPageInfo: PageInfo = {
  hasNextPage: false,
  nextCursor: null,
};

interface PaginatedState<T> {
  items: T[];
  pageInfo: PageInfo;
  loading: boolean;
  refreshing: boolean;
  loadingMore: boolean;
  error: string | null;
  errorDetail: RegistryRequestError | null;
}

interface UsePaginatedCollectionOptions<T, F extends { cursor?: string }> {
  loader: (filters: F) => Promise<SkillsPage<T>>;
  filters: F;
  deps: readonly unknown[];
  preserveStaleItemsOnRefresh?: boolean;
}

export function usePaginatedCollection<T, F extends { cursor?: string }>({
  loader,
  filters,
  deps,
  preserveStaleItemsOnRefresh = false,
}: UsePaginatedCollectionOptions<T, F>): PaginatedState<T> & { loadMore: () => void } {
  const [state, setState] = useState<PaginatedState<T>>({
    items: [],
    pageInfo: emptyPageInfo,
    loading: true,
    refreshing: false,
    loadingMore: false,
    error: null,
    errorDetail: null,
  });
  const requestVersionRef = useRef(0);

  useEffect(() => {
    let cancelled = false;
    requestVersionRef.current += 1;
    const requestVersion = requestVersionRef.current;

    setState((current) => ({
      items: preserveStaleItemsOnRefresh ? current.items : [],
      pageInfo: emptyPageInfo,
      loading: !preserveStaleItemsOnRefresh || current.items.length === 0,
      refreshing: preserveStaleItemsOnRefresh && current.items.length > 0,
      loadingMore: false,
      error: null,
      errorDetail: null,
    }));

    loader({ ...filters, cursor: undefined })
      .then((page) => {
        if (cancelled || requestVersionRef.current !== requestVersion) {
          return;
        }
        startTransition(() => {
          setState({
            items: page.items,
            pageInfo: page.pageInfo,
            loading: false,
            refreshing: false,
            loadingMore: false,
            error: null,
            errorDetail: null,
          });
        });
      })
      .catch((error: unknown) => {
        if (cancelled || requestVersionRef.current !== requestVersion) {
          return;
        }
        const normalizedError = normalizeThrownRegistryError(error);
        startTransition(() => {
          setState((current) => ({
            items: preserveStaleItemsOnRefresh ? current.items : [],
            pageInfo: emptyPageInfo,
            loading: false,
            refreshing: false,
            loadingMore: false,
            error: normalizedError.message,
            errorDetail: normalizedError,
          }));
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
    const requestVersion = requestVersionRef.current;

    setState((current) => ({
      ...current,
      loadingMore: true,
      error: null,
      errorDetail: null,
    }));

    loader({ ...filters, cursor: nextCursor })
      .then((page) => {
        if (requestVersionRef.current !== requestVersion) {
          return;
        }
        startTransition(() => {
          setState((current) => ({
            items: [...current.items, ...page.items],
            pageInfo: page.pageInfo,
            loading: false,
            refreshing: false,
            loadingMore: false,
            error: null,
            errorDetail: null,
          }));
        });
      })
      .catch((error: unknown) => {
        if (requestVersionRef.current !== requestVersion) {
          return;
        }
        const normalizedError = normalizeThrownRegistryError(error);
        startTransition(() => {
          setState((current) => ({
            ...current,
            loadingMore: false,
            error: normalizedError.message,
            errorDetail: normalizedError,
          }));
        });
      });
  }

  return {
    ...state,
    loadMore,
  };
}
