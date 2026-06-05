import type { PropsWithChildren } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { getRegistryClientConfig } from '../api/client';
import { canMutateCatalog, hasAction } from '../api/session';
import { useRegistrySession } from '../session/SessionProvider';

function isStaticRuntimeAuth() {
  const config = getRegistryClientConfig();
  return config.mode === 'mock' || config.hasToken || Boolean(config.role);
}

export function RequireAuthenticatedSession({ children }: PropsWithChildren) {
  const location = useLocation();
  const { session } = useRegistrySession();

  if (isStaticRuntimeAuth()) {
    return <>{children}</>;
  }

  if (session.loading) {
    return <div className="empty-box">正在加载会话…</div>;
  }

  if (session.errorDetail && session.errorDetail.kind !== 'auth') {
    return <div className="error-box">{session.error}</div>;
  }

  if (!session.data) {
    const next = `${location.pathname}${location.search}`;
    return <Navigate replace to={`/login?next=${encodeURIComponent(next)}`} />;
  }

  return <>{children}</>;
}

export function RequireOperatorSession({
  children,
  requiredAction,
}: PropsWithChildren<{ requiredAction?: string }>) {
  const { session } = useRegistrySession();

  if (session.loading) {
    return <div className="empty-box">正在加载会话…</div>;
  }

  if (!session.data) {
    return <Navigate replace to="/" />;
  }

  const allowed = requiredAction
    ? hasAction(session.data, requiredAction)
    : canMutateCatalog(session.data);

  if (!allowed) {
    return <Navigate replace to="/" />;
  }

  return <>{children}</>;
}
