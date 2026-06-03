import { Link } from 'react-router-dom';
import { listAuditEvents, listCuratedSkills, listGovernanceChanges, listSkills } from '../api/client';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { TimelineList } from '../components/TimelineList';
import { useAsyncData } from '../lib/useAsyncData';
import { formatDate } from '../lib/format';

export function OverviewPage() {
  const skills = useAsyncData(() => listSkills({ includeRetired: true, pageSize: 8 }), []);
  const curated = useAsyncData(() => listCuratedSkills(), []);
  const governance = useAsyncData(() => listGovernanceChanges({ pageSize: 5 }), []);
  const audit = useAsyncData(() => listAuditEvents({ pageSize: 5 }), []);
  const packagingEvents = audit.data?.items.filter((item) => item.action.startsWith('packaging.')) || [];
  const shownSkills = skills.data?.items.length || 0;
  const shownGovernance = governance.data?.items.length || 0;
  const shownAudit = audit.data?.items.length || 0;

  return (
    <div className="page-grid">
      <SectionHeading
        actions={
          <div className="authoring-actions">
            <Link className="button button--primary" to="/authoring">
              Open authoring flow
            </Link>
            <Link className="button button--secondary" to="/catalog">
              Browse catalog
            </Link>
          </div>
        }
        eyebrow="Registry Core MVP"
        title="把 skill discovery、release review 和 governance 串成一个像产品的入口。"
        description="前台先讲清楚 registry 的主线和节奏，后台再承接 import、发布和治理动作。这一屏负责把系统从“有页面”抬到“有方向感”。"
        tone="frontstage"
      />

      <div className="spotlight-grid">
        <SurfaceCard variant="hero">
          <div className="hero-kicker">Registry, Packaging, Release, Governance</div>
          <h2 className="hero-title">让作者从外部 skill intake 一路走到 release review，而不是在多个工具之间跳转。</h2>
          <p className="hero-copy">
            首页先回答两件事：这个产品到底管什么，以及当前下一步应该去哪里。前台做强发现和叙事，后台负责把动作收敛成可执行控制台。
          </p>
          <div className="authoring-actions">
            <Link className="button button--primary" to="/import">
              Start import
            </Link>
            <Link className="button button--secondary" to="/governance">
              Review governance
            </Link>
            <Link className="button button--secondary" to="/audit">
              Inspect audit
            </Link>
          </div>

          <div className="hero-band">
            <article className="hero-band__item">
              <span>Catalog footprint</span>
              <strong>{shownSkills || '—'}</strong>
              <p>{skills.data?.pageInfo.hasNextPage ? '这里只展示第一页 package 摘要。' : '当前 package 摘要已完整展开。'}</p>
            </article>
            <article className="hero-band__item">
              <span>Curated lane</span>
              <strong>{curated.data?.length || '—'}</strong>
              <p>当前值得被优先展示和试装的高价值 skill。</p>
            </article>
            <article className="hero-band__item">
              <span>Packaging motion</span>
              <strong>{packagingEvents.length || '—'}</strong>
              <p>最近是否真的有人在持续 intake、validate 和 pack。</p>
            </article>
          </div>
        </SurfaceCard>

        <SurfaceCard
          description="把运营风险和控制台语义先放在侧边，不跟主 hero 抢同一级重心。"
          title="Operator Context"
          variant="rail"
        >
          <div className="signal-list">
            <article className="signal-row">
              <strong>{shownGovernance || '—'}</strong>
              <div>
                <span>Governance pressure</span>
                <p>动作持续增长时，产品面需要更强调 retire、deprecate、yank 的影响和替代路径。</p>
              </div>
            </article>
            <article className="signal-row">
              <strong>{shownAudit || '—'}</strong>
              <div>
                <span>Audit visibility</span>
                <p>一旦流程进入发布或治理阶段，谁做了什么必须能被直接追溯。</p>
              </div>
            </article>
            <article className="signal-row">
              <strong>{packagingEvents.length || '—'}</strong>
              <div>
                <span>Workflow heat</span>
                <p>这不是只读 catalog，打包链路应该始终能感知到活跃度和失败面。</p>
              </div>
            </article>
          </div>
        </SurfaceCard>
      </div>

      <div className="metric-grid">
        <div className="metric-card">
          <strong>{shownSkills || '—'}</strong>
          <span>Indexed skills shown</span>
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
          <strong>{shownGovernance || '—'}</strong>
          <span>Governance shown</span>
        </div>
      </div>

      <div className="two-column">
        <SurfaceCard
          action={
            <Link className="button button--secondary" to="/catalog">
              Open catalog
            </Link>
          }
          description="首页不只展示 feed，也要给出最值得点开的 discovery 入口。"
          title="Curated Skills"
          variant="spotlight"
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
          <SurfaceCard description="治理变化应该被当成产品层事件，而不是埋在角落日志里。" title="Recent Governance" variant="console">
            <TimelineList
              items={(governance.data?.items || []).map((item) => ({
                key: item.id,
                title: `${item.action} ${item.packageName}${item.version ? ` ${item.version}` : ''}`,
                subtitle: `${item.reason} · ${item.changedBy} · ${formatDate(item.changedAt)}`,
                meta: item.replacementRecommendation ? (
                  <StatusPill tone="accent">replacement {item.replacementRecommendation}</StatusPill>
                ) : null,
              }))}
            />
          </SurfaceCard>
          <SurfaceCard description="审计记录回答谁在什么时候做了哪一步，以及结果是否真的落地。" title="Recent Audit" variant="console">
            <TimelineList
              items={(audit.data?.items || []).map((item) => ({
                key: item.id,
                title: item.action,
                subtitle: `${item.actor} · ${item.statusCode} · ${formatDate(item.createdAt)}`,
                meta: <StatusPill tone={item.outcome === 'succeeded' ? 'good' : 'bad'}>{item.outcome}</StatusPill>,
              }))}
            />
            {audit.data?.pageInfo.hasNextPage ? (
              <p className="support-copy">Overview 只显示第一页审计摘要；完整列表请去 Audit 页面继续翻页。</p>
            ) : null}
          </SurfaceCard>
        </div>
      </div>
    </div>
  );
}
