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
  sessionLoading,
  sessionError,
  requiredAction,
  packageName,
  fallback,
  unavailableFallback,
  children,
}: PermissionGateProps) {
  const registrySession = useOptionalRegistrySession();
  const resolvedSession =
    session !== undefined ? session : (registrySession?.session.data ?? null);
  const resolvedLoading =
    sessionLoading !== undefined
      ? sessionLoading
      : (registrySession?.session.loading ?? false);
  const resolvedError =
    sessionError !== undefined ? sessionError : (registrySession?.session.error ?? null);
  const posture = getSessionPosture(resolvedSession, {
    loading: resolvedLoading,
    error: resolvedError,
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

  const allowed = sessionHasCapability(resolvedSession, requiredAction, packageName);

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
