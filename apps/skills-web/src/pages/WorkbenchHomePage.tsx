import { Link } from 'react-router-dom';
import { listAuditEvents, listGovernanceChanges, listSkills } from '../api/client';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { useAsyncData } from '../lib/useAsyncData';

export function WorkbenchHomePage() {
  const skills = useAsyncData(() => listSkills({ includeRetired: true, pageSize: 8 }), []);
  const governance = useAsyncData(() => listGovernanceChanges({ pageSize: 8 }), []);
  const audit = useAsyncData(() => listAuditEvents({ pageSize: 8 }), []);
  const packagingEvents = audit.data?.items.filter((item) => item.action.startsWith('packaging.')) || [];
  const releaseEvents = audit.data?.items.filter((item) => item.action === 'release.record') || [];

  return (
    <div className="page-grid">
      <SectionHeading
        actions={
          <div className="authoring-actions">
            <Link className="button button--primary" to="/workbench/import">
              导入
            </Link>
            <Link className="button button--secondary" to="/workbench/authoring">
              作者台
            </Link>
          </div>
        }
        eyebrow="工作台"
        title="操作工作台"
        description="导入、发布审阅、治理和审计。"
        tone="backstage"
      />

      <div className="metric-grid">
        <div className="metric-card">
          <strong>{skills.data?.items.length || '—'}</strong>
          <span>技能包</span>
        </div>
        <div className="metric-card">
          <strong>{packagingEvents.length || '—'}</strong>
          <span>打包</span>
        </div>
        <div className="metric-card">
          <strong>{releaseEvents.length || '—'}</strong>
          <span>发布登记</span>
        </div>
        <div className="metric-card">
          <strong>{governance.data?.items.length || '—'}</strong>
          <span>治理</span>
        </div>
      </div>

      <div className="two-column">
        <SurfaceCard title="流程" variant="console">
          <div className="list-stack">
            <article className="list-card">
              <div className="list-card__head">
                <div>
                  <strong>导入</strong>
                  <span>准备、校验、打包</span>
                </div>
                <StatusPill tone="accent">{packagingEvents.length || 0}</StatusPill>
              </div>
              <div className="list-card__actions">
                <Link className="button button--secondary" to="/workbench/import">
                  打开
                </Link>
              </div>
            </article>
            <article className="list-card">
              <div className="list-card__head">
                <div>
                  <strong>作者台</strong>
                  <span>发布交接</span>
                </div>
                <StatusPill>{releaseEvents.length || 0}</StatusPill>
              </div>
              <div className="list-card__actions">
                <Link className="button button--secondary" to="/workbench/authoring">
                  打开
                </Link>
              </div>
            </article>
            <article className="list-card">
              <div className="list-card__head">
                <div>
                  <strong>治理</strong>
                  <span>退役、弃用、撤回</span>
                </div>
                <StatusPill tone="warn">{governance.data?.items.length || 0}</StatusPill>
              </div>
              <div className="list-card__actions">
                <Link className="button button--secondary" to="/workbench/governance">
                  打开
                </Link>
              </div>
            </article>
            <article className="list-card">
              <div className="list-card__head">
                <div>
                  <strong>审计</strong>
                  <span>最近追踪</span>
                </div>
                <StatusPill>{audit.data?.items.length || 0}</StatusPill>
              </div>
              <div className="list-card__actions">
                <Link className="button button--secondary" to="/workbench/audit">
                  打开
                </Link>
              </div>
            </article>
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
