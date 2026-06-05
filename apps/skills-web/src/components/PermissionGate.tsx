import type { PropsWithChildren, ReactNode } from 'react';
import type { RegistrySession } from '../api/types';
import {
  getSessionPosture,
  hasCapability as sessionHasCapability,
} from '../session/posture';
import { useOptionalRegistrySession } from '../session/SessionProvider';

type PermissionGateProps = PropsWithChildren<{
  session?: RegistrySession | null;
  sessionLoading?: boolean;
  sessionError?: string | null;
  requiredAction: string;
  packageName?: string | null;
  fallback?: ReactNode;
  unavailableFallback?: ReactNode;
}>;

export function PermissionGate({
  session,
  sessionLoading = false,
  sessionError = null,
  requiredAction,
  packageName,
  fallback,
  unavailableFallback,
  children,
}: PermissionGateProps) {
  const registrySession = useOptionalRegistrySession();
  const useProviderState =
    registrySession &&
    session === undefined &&
    sessionLoading === undefined &&
    sessionError === undefined;
  const resolvedSession = session === undefined ? (registrySession?.session.data ?? null) : session;
  const posture = useProviderState
    ? registrySession.getPosture({ packageName, requiredAction })
    : getSessionPosture(resolvedSession, {
        loading: sessionLoading ?? false,
        error: sessionError ?? null,
        packageName,
        requiredAction,
      });

  if (posture === 'loading') {
    return <>{unavailableFallback ?? <div className="empty-box">正在加载会话权限…</div>}</>;
  }

  if (posture === 'unavailable') {
    return (
      <>
        {unavailableFallback ?? (
          <div className="empty-box">当前会话不可用，暂时无法校验 {requiredAction}。</div>
        )}
      </>
    );
  }

  if (posture === 'unauthenticated') {
    return <>{unavailableFallback ?? <div className="empty-box">当前没有可用于 {requiredAction} 的已登录 Catalog 会话。</div>}</>;
  }

  const allowed = useProviderState
    ? registrySession.hasCapability(requiredAction, packageName)
    : sessionHasCapability(resolvedSession, requiredAction, packageName);

  if (!allowed) {
    return (
      <>
        {fallback ?? (
          <div className="empty-box">
            当前会话没有执行 {requiredAction}
            {packageName?.trim() ? `（${packageName.trim()}）` : ''}
            的权限。
          </div>
        )}
      </>
    );
  }

  return <>{children}</>;
}
