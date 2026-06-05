import { createContext, type PropsWithChildren, useContext, useState } from 'react';
import { getRegistryClientConfig, getSession } from '../api/client';
import type { RegistrySession } from '../api/types';
import { useAsyncData, type AsyncState } from '../lib/useAsyncData';

type SessionContextValue = {
  session: AsyncState<RegistrySession | null>;
  refreshSession: () => void;
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
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}

export function useRegistrySession() {
  const value = useContext(SessionContext);
  if (!value) {
    throw new Error('useRegistrySession must be used inside SessionProvider');
  }
  return value;
}
