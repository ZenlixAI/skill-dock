import type { PropsWithChildren, ReactNode } from 'react';
import { canPerformAction } from '../api/session';
import type { RegistrySession } from '../api/types';

type PermissionGateProps = PropsWithChildren<{
  session: RegistrySession | null;
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
  if (sessionLoading) {
    return <>{unavailableFallback ?? <div className="empty-box">正在加载会话权限…</div>}</>;
  }

  if (sessionError) {
    return (
      <>
        {unavailableFallback ?? (
          <div className="empty-box">当前会话不可用，暂时无法校验 {requiredAction}。</div>
        )}
      </>
    );
  }

  if (!session) {
    return <>{unavailableFallback ?? <div className="empty-box">当前没有可用于 {requiredAction} 的已登录 Catalog 会话。</div>}</>;
  }

  const allowed = canPerformAction(session, requiredAction, packageName);

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
