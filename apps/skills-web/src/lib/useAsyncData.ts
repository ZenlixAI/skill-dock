import { startTransition, useEffect, useRef, useState } from 'react';
import { normalizeThrownRegistryError } from '../api/httpClient';
import type { RegistryRequestError } from '../api/types';

export interface AsyncState<T> {
  data: T | null;
  loading: boolean;
  refreshing: boolean;
  error: string | null;
  errorDetail: RegistryRequestError | null;
}

interface UseAsyncDataOptions {
  preserveStaleData?: boolean;
}

export function useAsyncData<T>(
  loader: () => Promise<T>,
  deps: readonly unknown[],
  options?: UseAsyncDataOptions
): AsyncState<T> {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: true,
    refreshing: false,
    error: null,
    errorDetail: null,
  });
  const requestVersionRef = useRef(0);

  useEffect(() => {
    let cancelled = false;
    requestVersionRef.current += 1;
    const requestVersion = requestVersionRef.current;
    const preserveStaleData = options?.preserveStaleData === true;

    setState((current) => ({
      data: preserveStaleData ? current.data : null,
      loading: !preserveStaleData || current.data == null,
      refreshing: preserveStaleData && current.data != null,
      error: null,
      errorDetail: null,
    }));

    loader()
      .then((data) => {
        if (cancelled || requestVersionRef.current !== requestVersion) {
          return;
        }
        startTransition(() => {
          setState({
            data,
            loading: false,
            refreshing: false,
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
            data: preserveStaleData ? current.data : null,
            loading: false,
            refreshing: false,
            error: normalizedError.message,
            errorDetail: normalizedError,
          }));
        });
      });

    return () => {
      cancelled = true;
    };
  }, deps);

  return state;
}
