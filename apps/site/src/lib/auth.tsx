import { LogtoProvider, useLogto } from '@logto/react';
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from 'react';

export interface AuthOrganization {
  id: string;
  logtoOrgId: string;
  slug: string;
  name: string;
  role: string;
}

export interface AuthUser {
  id: string;
  displayName?: string;
  email?: string;
}

export interface AuthState {
  enabled: boolean;
  loading: boolean;
  authenticated: boolean;
  user: AuthUser | null;
  organizations: AuthOrganization[];
  activeOrganization: AuthOrganization | null;
  apiBase: string;
  signIn: () => Promise<void>;
  signOut: () => Promise<void>;
  switchOrganization: (organizationId: string) => void;
  getAccessToken: () => Promise<string | undefined>;
  authFetch: (path: string, init?: RequestInit) => Promise<Response>;
}

const disabledAuth: AuthState = {
  enabled: false,
  loading: false,
  authenticated: false,
  user: null,
  organizations: [],
  activeOrganization: null,
  apiBase: '/api/v1',
  signIn: async () => {},
  signOut: async () => {},
  switchOrganization: () => {},
  getAccessToken: async () => undefined,
  authFetch: async (path, init) => fetch(path, init),
};

const AuthContext = createContext<AuthState>(disabledAuth);

export function useAuth(): AuthState {
  return useContext(AuthContext);
}

export function AuthProvider({ children }: PropsWithChildren) {
  const endpoint = (import.meta.env.VITE_LOGTO_ENDPOINT || '').trim();
  const appId = (import.meta.env.VITE_LOGTO_APP_ID || '').trim();
  const apiResource = (import.meta.env.VITE_LOGTO_API_RESOURCE || '').trim();
  const apiBase = (import.meta.env.VITE_CATALOG_API_URL || '/api/v1').replace(/\/+$/, '');

  if (!endpoint || !appId || !apiResource) {
    return <AuthContext.Provider value={{ ...disabledAuth, apiBase }}>{children}</AuthContext.Provider>;
  }

  return (
    <LogtoProvider
      config={{
        endpoint,
        appId,
        scopes: ['openid', 'profile', 'email', 'offline_access', 'catalog:read', 'catalog:install', 'favorites:read', 'favorites:write', 'urn:logto:scope:organizations'],
        resources: apiResource ? [apiResource] : undefined,
      }}
    >
      <AuthBridge apiBase={apiBase} apiResource={apiResource}>
        {children}
      </AuthBridge>
    </LogtoProvider>
  );
}

function AuthBridge({
  children,
  apiBase,
  apiResource,
}: PropsWithChildren<{ apiBase: string; apiResource: string }>) {
  const logto = useLogto();
  const [user, setUser] = useState<AuthUser | null>(null);
  const [organizations, setOrganizations] = useState<AuthOrganization[]>([]);
  const [activeOrganization, setActiveOrganization] = useState<AuthOrganization | null>(null);
  const [loading, setLoading] = useState(false);

  const getAccessToken = useCallback(async () => {
    if (!logto.isAuthenticated) return undefined;
    try {
      if (activeOrganization) {
        return await logto.getOrganizationToken(activeOrganization.logtoOrgId || activeOrganization.id);
      }
      return await logto.getAccessToken(apiResource || undefined);
    } catch {
      return undefined;
    }
  }, [activeOrganization, apiResource, logto]);

  const authFetch = useCallback(
    async (path: string, init?: RequestInit) => {
      const token = await getAccessToken();
      const headers = new Headers(init?.headers);
      if (token) headers.set('Authorization', `Bearer ${token}`);
      headers.set('accept', 'application/json');
      return fetch(path, { ...init, headers });
    },
    [getAccessToken]
  );

  useEffect(() => {
    if (!logto.isAuthenticated) {
      setUser(null);
      setOrganizations([]);
      setActiveOrganization(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        const token = await logto.getAccessToken(apiResource || undefined);
        const response = await fetch(`${apiBase}/me`, {
          headers: { Authorization: `Bearer ${token}`, accept: 'application/json' },
        });
        if (!response.ok) return;
        const payload = (await response.json()) as {
          user: AuthUser;
          organizations?: Array<{ id: string; logtoOrgId: string; slug: string; name: string; role: string }>;
        };
        if (cancelled) return;
        setUser(payload.user);
        const nextOrganizations = payload.organizations ?? [];
        setOrganizations(nextOrganizations);
        setActiveOrganization((current) => {
          if (current) return nextOrganizations.find((item) => item.id === current.id) ?? nextOrganizations[0] ?? null;
          return nextOrganizations[0] ?? null;
        });
      } catch {
        // Keep the local experience when the catalog is unavailable.
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
    // Logto exposes a stable client; the authenticated flag is the trigger.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiBase, apiResource, logto.isAuthenticated]);

  const value = useMemo<AuthState>(
    () => ({
      enabled: true,
      loading: loading || logto.isLoading,
      authenticated: logto.isAuthenticated,
      user,
      organizations,
      activeOrganization,
      apiBase,
      signIn: async () => {
        sessionStorage.setItem('skill-dock:return-to', window.location.pathname + window.location.search);
        await logto.signIn(`${window.location.origin}/callback`);
      },
      signOut: async () => {
        sessionStorage.removeItem('skill-dock:return-to');
        await logto.signOut(`${window.location.origin}/`);
      },
      switchOrganization: (organizationId: string) => {
        setActiveOrganization(organizations.find((item) => item.id === organizationId) ?? null);
      },
      getAccessToken,
      authFetch,
    }),
    [activeOrganization, apiBase, authFetch, getAccessToken, loading, logto, organizations, user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
