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
  const isLoading = skills.loading || governance.loading || audit.loading;

  return (
    <div className="page-grid">
      <SectionHeading
        actions={
          <div className="authoring-actions">
            <Link className="button button--primary" to="/workbench/import">
              打开导入
            </Link>
            <Link className="button button--secondary" to="/workbench/authoring">
              打开作者台
            </Link>
          </div>
        }
        eyebrow="控制台"
        title="控制台首页"
        description="把导入、作者台、治理和审计流程收拢到次级操作台，避免挤占 Portal 的发现路径。"
        tone="backstage"
      />

      {isLoading ? (
        <div aria-hidden="true" className="console-skeleton-grid">
          <div className="portal-skeleton" />
          <div className="portal-skeleton" />
        </div>
      ) : null}

      <div className="metric-grid">
        <div className="metric-card">
          <strong>{skills.data?.items.length ?? '—'}</strong>
          <span>技能包</span>
        </div>
        <div className="metric-card">
          <strong>{packagingEvents.length}</strong>
          <span>打包事件</span>
        </div>
        <div className="metric-card">
          <strong>{releaseEvents.length}</strong>
          <span>发布交接</span>
        </div>
        <div className="metric-card">
          <strong>{governance.data?.items.length ?? '—'}</strong>
          <span>治理事件</span>
        </div>
      </div>

      <div className="console-home-grid">
        <SurfaceCard
          description="准备、校验、打包外部技能，或把作者台里的发布交接继续推进。"
          title="导入与打包"
          variant="console"
        >
          <div className="pill-row">
            <StatusPill tone="accent">{packagingEvents.length} 条打包事件</StatusPill>
            <StatusPill>{releaseEvents.length} 条发布交接</StatusPill>
          </div>
          <div className="list-card__actions">
            <Link className="button button--primary" to="/workbench/import">
              打开导入
            </Link>
            <Link className="button button--secondary" to="/workbench/authoring">
              打开作者台
            </Link>
          </div>
        </SurfaceCard>

        <SurfaceCard
          description="先看影响范围与追溯摘要，再进入治理与审计的明细事件流。"
          title="治理与审计"
          variant="console"
        >
          <div className="pill-row">
            <StatusPill tone="warn">{governance.data?.items.length ?? 0} 条治理事件</StatusPill>
            <StatusPill>{audit.data?.items.length ?? 0} 条审计事件</StatusPill>
          </div>
          <div className="list-card__actions">
            <Link className="button button--secondary" to="/workbench/governance">
              打开治理
            </Link>
            <Link className="button button--secondary" to="/workbench/audit">
              打开审计
            </Link>
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
