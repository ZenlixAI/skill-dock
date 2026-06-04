import type { RegistrySession, RegistryRuntimeMode } from '../api/types';
import { canMutateCatalog, describeSessionPosture, describeSessionScope } from '../api/session';

interface EnvironmentGuardProps {
  mode: RegistryRuntimeMode;
  baseUrl: string | null;
  session: RegistrySession | null;
  sessionError?: string | null;
}

export function EnvironmentGuard({ mode, baseUrl, session, sessionError }: EnvironmentGuardProps) {
  if (mode === 'mock') {
    return <div className="env-banner">Mock mode: current writes do not touch the real Catalog.</div>;
  }

  if (sessionError) {
    return <div className="env-banner">Real mode warning: failed to load session posture. {sessionError}</div>;
  }

  return (
    <div className="env-banner">
      <span>{baseUrl ? `connected to ${baseUrl}` : 'connected to Catalog API'}</span>
      <span>{describeSessionPosture(session)}</span>
      {describeSessionScope(session) ? <span>{describeSessionScope(session)}</span> : null}
      {canMutateCatalog(session) ? <span>mutations enabled</span> : <span>mutations restricted</span>}
    </div>
  );
}
