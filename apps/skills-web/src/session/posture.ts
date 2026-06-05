import type { RegistrySession } from '../api/types';
import { canPerformAction, canMutateCatalog, describeSessionPosture } from '../api/session';

export type SessionPosture =
  | 'loading'
  | 'unavailable'
  | 'unauthenticated'
  | 'read_only'
  | 'write_enabled'
  | 'package_filter_required';

export function isSessionUnavailable(loading: boolean, error: string | null) {
  return !loading && Boolean(error);
}

export function hasCapability(
  session: RegistrySession | null,
  action: string,
  packageName?: string | null
) {
  return canPerformAction(session, action, packageName);
}

export function getSessionPosture(
  session: RegistrySession | null,
  options: {
    loading?: boolean;
    error?: string | null;
    packageName?: string | null;
    requiredAction?: string;
  } = {}
): SessionPosture {
  if (options.loading) {
    return 'loading';
  }
  if (isSessionUnavailable(Boolean(options.loading), options.error ?? null)) {
    return 'unavailable';
  }
  if (!session) {
    return 'unauthenticated';
  }
  if (
    options.requiredAction
  ) {
    if (
      session.allowedActions.includes(options.requiredAction) &&
      !options.packageName?.trim() &&
      session.allowedPackages.length > 0
    ) {
      return 'package_filter_required';
    }
    return hasCapability(session, options.requiredAction, options.packageName)
      ? 'write_enabled'
      : 'read_only';
  }
  return canMutateCatalog(session) ? 'write_enabled' : 'read_only';
}

export function getSessionPostureLabel(
  session: RegistrySession | null,
  options: {
    loading?: boolean;
    error?: string | null;
    packageName?: string | null;
    requiredAction?: string;
  } = {}
) {
  const posture = getSessionPosture(session, options);
  if (posture === 'loading') return '会话加载中';
  if (posture === 'unavailable') return '会话不可用';
  if (posture === 'unauthenticated') return '未登录';
  if (posture === 'package_filter_required') return '需要先选定包名';
  return describeSessionPosture(session);
}
