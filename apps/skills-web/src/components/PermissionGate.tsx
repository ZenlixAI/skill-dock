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
    return <>{unavailableFallback ?? <div className="empty-box">Loading session permissions…</div>}</>;
  }

  if (sessionError) {
    return (
      <>
        {unavailableFallback ?? (
          <div className="empty-box">Session unavailable. Unable to verify {requiredAction} right now.</div>
        )}
      </>
    );
  }

  if (!session) {
    return <>{unavailableFallback ?? <div className="empty-box">No authenticated Catalog session for {requiredAction}.</div>}</>;
  }

  const allowed = canPerformAction(session, requiredAction, packageName);

  if (!allowed) {
    return (
      <>
        {fallback ?? (
          <div className="empty-box">
            Current session cannot perform {requiredAction}
            {packageName?.trim() ? ` for ${packageName.trim()}` : ''}.
          </div>
        )}
      </>
    );
  }

  return <>{children}</>;
}
