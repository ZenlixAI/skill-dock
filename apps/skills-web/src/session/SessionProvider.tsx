import { createContext, type PropsWithChildren, useContext, useState } from 'react';
import { getRegistryClientConfig, getSession } from '../api/client';
import type { RegistrySession } from '../api/types';
import { useAsyncData, type AsyncState } from '../lib/useAsyncData';
import {
  getSessionPosture,
  getSessionPostureLabel,
  hasCapability as sessionHasCapability,
  type SessionPosture,
} from './posture';

type SessionContextValue = {
  session: AsyncState<RegistrySession | null>;
  refreshSession: () => void;
  getPosture: (options?: { packageName?: string | null; requiredAction?: string }) => SessionPosture;
  getPostureLabel: (options?: { packageName?: string | null; requiredAction?: string }) => string;
  hasCapability: (action: string, packageName?: string | null) => boolean;
};

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: PropsWithChildren) {
  const config = getRegistryClientConfig();
  const [refreshVersion, setRefreshVersion] = useState(0);
  const session = useAsyncData(() => getSession(), [config.mode, config.baseUrl, refreshVersion]);

  return (
    <SessionContext.Provider
      value={{
        session,
        refreshSession: () => setRefreshVersion((current) => current + 1),
        getPosture: (options) =>
          getSessionPosture(session.data, {
            loading: session.loading,
            error: session.error,
            ...options,
          }),
        getPostureLabel: (options) =>
          getSessionPostureLabel(session.data, {
            loading: session.loading,
            error: session.error,
            ...options,
          }),
        hasCapability: (action, packageName) =>
          sessionHasCapability(session.data, action, packageName),
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}

export function useOptionalRegistrySession() {
  return useContext(SessionContext);
}

export function useRegistrySession() {
  const value = useOptionalRegistrySession();
  if (!value) {
    throw new Error('useRegistrySession must be used inside SessionProvider');
  }
  return value;
}
