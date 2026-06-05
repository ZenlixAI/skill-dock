import { Link } from 'react-router-dom';
import { listAuditEvents, listCuratedSkills, listGovernanceChanges, listSkills } from '../api/client';
import type { RegistryRequestError } from '../api/types';
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
  const shownSkills = skills.data?.items.length || 0;
  const pageError = [skills.errorDetail, curated.errorDetail, governance.errorDetail, audit.errorDetail].find(Boolean) || null;

  return (
    <div className="page-grid">
      <SectionHeading
        actions={
          <div className="authoring-actions">
            <Link className="button button--primary" to="/catalog">
              浏览目录
            </Link>
          </div>
        }
        eyebrow="首页"
        title="发现技能"
        tone="frontstage"
      />

      {pageError ? <div className="error-box">{describeOverviewError(pageError)}</div> : null}

      <div className="spotlight-grid">
        <SurfaceCard variant="hero">
          <div className="hero-kicker">目录</div>
          <h2 className="hero-title">浏览精选技能和最近变化。</h2>
          <div className="hero-band">
            <article className="hero-band__item">
              <span>目录</span>
              <strong>{shownSkills || '—'}</strong>
              <p>{skills.data?.pageInfo.hasNextPage ? '已加载第一页。' : '当前页已加载。'}</p>
            </article>
            <article className="hero-band__item">
              <span>精选</span>
              <strong>{curated.data?.length || '—'}</strong>
              <p>优先展示的技能。</p>
            </article>
          </div>
        </SurfaceCard>
      </div>

      <div className="two-column">
        <SurfaceCard
          action={
            <Link className="button button--secondary" to="/catalog">
              打开目录
            </Link>
          }
          title="精选技能"
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
            {!curated.loading && !curated.errorDetail && curated.data?.length === 0 ? (
              <div className="empty-box">当前没有可展示的精选技能。</div>
            ) : null}
          </div>
        </SurfaceCard>

        <div className="stack-column">
          <SurfaceCard title="最近治理" variant="console">
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
            {!governance.loading && !governance.errorDetail && governance.data?.items.length === 0 ? (
              <div className="empty-box">当前还没有治理记录。</div>
            ) : null}
          </SurfaceCard>
          <SurfaceCard title="最近审计" variant="console">
            <TimelineList
              items={(audit.data?.items || []).map((item) => ({
                key: item.id,
                title: item.action,
                subtitle: `${item.actor} · ${item.statusCode} · ${formatDate(item.createdAt)}`,
                meta: <StatusPill tone={item.outcome === 'succeeded' ? 'good' : 'bad'}>{item.outcome}</StatusPill>,
              }))}
            />
            {!audit.loading && !audit.errorDetail && audit.data?.items.length === 0 ? (
              <div className="empty-box">当前还没有审计记录。</div>
            ) : null}
          </SurfaceCard>
        </div>
      </div>
    </div>
  );
}

function describeOverviewError(error: RegistryRequestError) {
  if (error.kind === 'auth') {
    return `${error.message} 当前会话无法读取部分首页数据。`;
  }

  if (error.kind === 'network' || error.kind === 'server') {
    return `${error.message} 首页数据未能完整加载。`;
  }

  if (error.kind === 'not_found') {
    return `${error.message} 加载过程中有一组数据发生了变化。`;
  }

  return error.message;
}
