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
    return <div className="env-banner">模拟模式：当前写操作不会落到真实 Catalog。</div>;
  }

  if (sessionError) {
    return <div className="env-banner">真实模式告警：会话态加载失败。{sessionError}</div>;
  }

  return (
    <div className="env-banner">
      <span>{baseUrl ? `已连接 ${baseUrl}` : '已连接 Catalog API'}</span>
      <span>{describeSessionPosture(session)}</span>
      {describeSessionScope(session) ? <span>{describeSessionScope(session)}</span> : null}
      {canMutateCatalog(session) ? <span>允许写操作</span> : <span>仅可只读</span>}
    </div>
  );
}
