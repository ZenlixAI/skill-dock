import { startTransition, useEffect, useState } from 'react';

export interface AsyncState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

export function useAsyncData<T>(
  loader: () => Promise<T>,
  deps: readonly unknown[]
): AsyncState<T> {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let cancelled = false;

    setState((current) => ({
      data: current.data,
      loading: true,
      error: null,
    }));

    loader()
      .then((data) => {
        if (cancelled) {
          return;
        }
        startTransition(() => {
          setState({
            data,
            loading: false,
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
            data: null,
            loading: false,
            error: message,
          });
        });
      });

    return () => {
      cancelled = true;
    };
  }, deps);

  return state;
}
