import { Link } from 'react-router-dom';
import { listAuditEvents, listCuratedSkills, listGovernanceChanges, listSkills } from '../api/client';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { TimelineList } from '../components/TimelineList';
import { useAsyncData } from '../lib/useAsyncData';
import { formatDate } from '../lib/format';

export function OverviewPage() {
  const skills = useAsyncData(() => listSkills({ includeRetired: true }), []);
  const curated = useAsyncData(() => listCuratedSkills(), []);
  const governance = useAsyncData(() => listGovernanceChanges(), []);
  const audit = useAsyncData(() => listAuditEvents(), []);
  const packagingEvents = audit.data?.filter((item) => item.action.startsWith('packaging.')) || [];

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Registry Core MVP"
        title="把 Catalog、Packaging、Governance 收成一个能用的 Web 工作台。"
        description="这不是 marketing 壳子，而是围绕当前 Go Catalog、skill packaging 和 release/governance 契约搭出的首版产品入口。"
      />

      <div className="spotlight-grid">
        <SurfaceCard className="surface-card--hero">
          <div className="hero-kicker">Registry, Packaging, Release, Governance</div>
          <h2 className="hero-title">让作者从外部 skill intake 一路走到 release review，而不是在多个工具之间跳转。</h2>
          <p className="hero-copy">
            首页应该先告诉人这套系统的主线是什么，再告诉人下一步去哪。这里优先突出 catalog 资产面、authoring flow 和最近发生的治理事件。
          </p>
          <div className="authoring-actions">
            <Link className="button button--primary" to="/authoring">
              Open authoring flow
            </Link>
            <Link className="button button--secondary" to="/catalog">
              Browse catalog
            </Link>
            <Link className="button button--secondary" to="/import">
              Start import
            </Link>
          </div>
        </SurfaceCard>

        <SurfaceCard className="surface-card--rail" title="Operator Notes" description="把系统当前最重要的三个观察先露出来。">
          <div className="signal-list">
            <article className="signal-row">
              <strong>{skills.data?.length || '—'}</strong>
              <div>
                <span>Catalog footprint</span>
                <p>当前 registry 里有多少 package 可以被装、被治理、被追溯。</p>
              </div>
            </article>
            <article className="signal-row">
              <strong>{packagingEvents.length || '—'}</strong>
              <div>
                <span>Packaging motion</span>
                <p>看当前是不是有人在持续做导入、验证、打包，而不是只停留在只读 catalog。</p>
              </div>
            </article>
            <article className="signal-row">
              <strong>{governance.data?.length || '—'}</strong>
              <div>
                <span>Governance pressure</span>
                <p>如果这里持续增长，说明产品面要更强调 yank / deprecate / retire 的后果和替代建议。</p>
              </div>
            </article>
          </div>
        </SurfaceCard>
      </div>

      <div className="metric-grid">
        <div className="metric-card">
          <strong>{skills.data?.length || '—'}</strong>
          <span>Indexed skills</span>
        </div>
        <div className="metric-card">
          <strong>{curated.data?.length || '—'}</strong>
          <span>Curated skills</span>
        </div>
        <div className="metric-card">
          <strong>{packagingEvents.length || '—'}</strong>
          <span>Packaging events</span>
        </div>
        <div className="metric-card">
          <strong>{governance.data?.length || '—'}</strong>
          <span>Governance actions today</span>
        </div>
      </div>

      <div className="two-column">
        <SurfaceCard
          title="Curated Skills"
          description="首页不应该只堆时间线，还是要给人一个高价值入口。这里优先露出最值得点开的 skill。"
          action={
            <div className="authoring-actions">
              <Link className="button button--secondary" to="/catalog">
                Open catalog
              </Link>
              <Link className="button button--secondary" to="/authoring">
                Open authoring flow
              </Link>
            </div>
          }
        >
          <div className="list-stack">
            {curated.data?.map((skill) => (
              <Link className="list-card list-card--feature" key={skill.packageName} to={`/skills/${encodeURIComponent(skill.packageName)}`}>
                <div className="list-card__head">
                  <div>
                    <strong>{skill.displayName}</strong>
                    <span>{skill.packageName}</span>
                  </div>
                  <StatusPill tone={skill.latestVersionStatus === 'published' ? 'good' : 'warn'}>
                    {skill.latestVersionStatus}
                  </StatusPill>
                </div>
                <p>{skill.description}</p>
                <div className="pill-row">
                  <StatusPill tone="accent">{skill.sourceType}</StatusPill>
                  <StatusPill tone="good">latest {skill.latestVersion}</StatusPill>
                </div>
              </Link>
            ))}
          </div>
        </SurfaceCard>

        <div className="stack-column">
          <SurfaceCard title="Recent Governance" description="治理变化不做藏起来的日志，而是明确暴露为产品层事件。">
            <TimelineList
              items={(governance.data || []).map((item) => ({
                key: item.id,
                title: `${item.action} ${item.packageName}${item.version ? ` ${item.version}` : ''}`,
                subtitle: `${item.reason} · ${item.changedBy} · ${formatDate(item.changedAt)}`,
                meta: item.replacementRecommendation ? (
                  <StatusPill tone="accent">replacement {item.replacementRecommendation}</StatusPill>
                ) : null,
              }))}
            />
          </SurfaceCard>
          <SurfaceCard title="Recent Audit" description="审计记录回答‘谁在什么时候做了哪一步’。">
            <TimelineList
              items={(audit.data || []).map((item) => ({
                key: item.id,
                title: item.action,
                subtitle: `${item.actor} · ${item.statusCode} · ${formatDate(item.createdAt)}`,
                meta: <StatusPill tone={item.outcome === 'succeeded' ? 'good' : 'bad'}>{item.outcome}</StatusPill>,
              }))}
            />
          </SurfaceCard>
        </div>
      </div>
    </div>
  );
}
