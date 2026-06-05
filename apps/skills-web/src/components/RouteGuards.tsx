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
  const { session, getPosture } = useRegistrySession();
  const posture = getPosture();

  if (isStaticRuntimeAuth()) {
    return <>{children}</>;
  }

  if (posture === 'loading') {
    return <div className="empty-box">正在加载会话…</div>;
  }

  if (posture === 'unavailable') {
    if (session.errorDetail?.kind === 'auth') {
      const next = `${location.pathname}${location.search}`;
      return <Navigate replace to={`/login?next=${encodeURIComponent(next)}`} />;
    }
    return <div className="error-box">{session.error}</div>;
  }

  if (posture === 'unauthenticated') {
    const next = `${location.pathname}${location.search}`;
    return <Navigate replace to={`/login?next=${encodeURIComponent(next)}`} />;
  }

  return <>{children}</>;
}

export function RequireOperatorSession({
  children,
  requiredAction,
}: PropsWithChildren<{ requiredAction?: string }>) {
  const { session, getPosture } = useRegistrySession();
  const posture = getPosture();

  if (posture === 'loading') {
    return <div className="empty-box">正在加载会话…</div>;
  }

  if (posture === 'unauthenticated' || !session.data) {
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
