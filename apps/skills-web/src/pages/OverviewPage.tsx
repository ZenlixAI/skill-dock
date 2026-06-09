import { startTransition, type FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { listAuditEvents, listCuratedSkills, listGovernanceChanges, listSkills } from '../api/client';
import type { RegistryRequestError } from '../api/types';
import { SectionHeading } from '../components/SectionHeading';
import { SkillSummaryCard } from '../components/SkillSummaryCard';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { TimelineList } from '../components/TimelineList';
import { formatDate } from '../lib/format';
import { useAsyncData } from '../lib/useAsyncData';

export function OverviewPage() {
  const [query, setQuery] = useState('');
  const navigate = useNavigate();
  const skills = useAsyncData(() => listSkills({ includeRetired: true, pageSize: 8 }), []);
  const curated = useAsyncData(() => listCuratedSkills(), []);
  const governance = useAsyncData(() => listGovernanceChanges({ pageSize: 5 }), []);
  const audit = useAsyncData(() => listAuditEvents({ pageSize: 5 }), []);
  const shownSkills = skills.data?.items.length || 0;
  const recentSkills = [...(skills.data?.items || [])]
    .sort((left, right) => Date.parse(right.updatedAt) - Date.parse(left.updatedAt))
    .slice(0, 3);
  const loadingDiscovery = skills.loading || curated.loading;
  const pageError = [skills.errorDetail, curated.errorDetail, governance.errorDetail, audit.errorDetail].find(Boolean) || null;

  function handleSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const next = query.trim();

    startTransition(() => {
      navigate(next ? `/catalog?q=${encodeURIComponent(next)}` : '/catalog');
    });
  }

  return (
    <div className="page-grid">
      <SectionHeading
        actions={
          <div className="context-strip">
            <StatusPill tone="accent">门户</StatusPill>
            <Link className="button button--primary" to="/catalog">
              查看目录
            </Link>
          </div>
        }
        description="通过门户搜索能力、评估包来源，并进入可直接安装的技能详情页。"
        eyebrow="概览"
        title="从技能注册表开始发现。"
        tone="frontstage"
      />

      <SurfaceCard className="portal-hero" variant="hero">
        <div className="hero-kicker">发现门户</div>
        <h2 className="hero-title">先按能力检索，再查看安装状态和包来源。</h2>
        <p className="hero-copy">
          推荐技能会保留在近处，最近更新和控制台流程也仍然可达，同时避免把落地页变成运营面板。
        </p>

        <form className="portal-hero__search" onSubmit={handleSearchSubmit}>
          <label className="field">
            <span>搜索技能</span>
            <input
              onChange={(event) => setQuery(event.target.value)}
              placeholder="评审、pdf、治理、导入"
              value={query}
            />
          </label>
          <button className="button button--primary" type="submit">
            搜索目录
          </button>
        </form>

        <div className="hero-band">
          <article className="hero-band__item">
            <span>目录切片</span>
            <strong>{skills.loading ? '—' : shownSkills}</strong>
            <p>{skills.data?.pageInfo.hasNextPage ? '当前展示首个发现切片。' : '当前展示这一发现切片。'}</p>
          </article>
          <article className="hero-band__item">
            <span>推荐</span>
            <strong>{curated.loading ? '—' : curated.data?.length || 0}</strong>
            <p>适合常见内部流程的精选技能。</p>
          </article>
          <article className="hero-band__item">
            <span>控制台衔接</span>
            <strong>{governance.data?.items.length || audit.data?.items.length ? '实时' : '就绪'}</strong>
            <p>当发现需要更深的运营上下文时，治理和审计仍然只差一步。</p>
          </article>
        </div>
      </SurfaceCard>

      {pageError ? <div className="error-box">{describeOverviewError(pageError)}</div> : null}

      {loadingDiscovery ? (
        <div aria-hidden="true" className="portal-skeleton-grid">
          <div className="portal-skeleton portal-skeleton--hero" />
          <div className="portal-skeleton" />
          <div className="portal-skeleton" />
        </div>
      ) : null}

      <div className="two-column">
        <SurfaceCard
          action={
            <Link className="button button--secondary" to="/catalog">
              查看全部技能
            </Link>
          }
          description="推荐项优先强调用途，同时保留可直接安装的元数据。"
          title="推荐技能"
          variant="spotlight"
        >
          <div className="list-stack">
            {curated.data?.map((skill) => (
              <SkillSummaryCard curationReason={skill.curationReason} key={skill.packageName} skill={skill} />
            ))}
            {!curated.loading && !curated.errorDetail && curated.data?.length === 0 ? (
              <div className="empty-box">
                <strong>当前还没有已发布的推荐技能。</strong>
                <p>在推荐整理完成前，可先通过目录浏览完整注册表。</p>
              </div>
            ) : null}
          </div>
        </SurfaceCard>

        <div className="stack-column">
          <SurfaceCard className="console-entry-card" description="当发现进入交付阶段时，可转到导入、治理和审计流程。" title="控制台" variant="subtle">
            <div className="console-entry-card__actions">
              <Link className="button button--secondary" to="/workbench">
                打开控制台
              </Link>
              <Link className="button button--secondary" to="/workbench/audit">
                查看审计
              </Link>
            </div>
          </SurfaceCard>

          <SurfaceCard
            action={
              <Link className="button button--secondary" to="/catalog">
                查看结果
              </Link>
            }
            description="最近更新的技能让发现保持新鲜，不必先进入控制台。"
            title="最近更新"
            variant="spotlight"
          >
            <div className="list-stack">
              {recentSkills.map((skill) => (
                <SkillSummaryCard key={skill.packageName} skill={skill} />
              ))}
              {!skills.loading && !skills.errorDetail && recentSkills.length === 0 ? (
                <div className="empty-box">
                  <strong>当前没有最近更新。</strong>
                  <p>可以先浏览更广的目录，或在新版本发布后再回来查看。</p>
                </div>
              ) : null}
            </div>
          </SurfaceCard>

          <SurfaceCard description="最近治理变更有助于解释包状态为何发生变化。" title="最新治理" variant="console">
            <TimelineList
              items={(governance.data?.items || []).map((item) => ({
                key: item.id,
                title: `${item.action} ${item.packageName}${item.version ? ` ${item.version}` : ''}`,
                subtitle: `${item.reason} · ${item.changedBy} · ${formatDate(item.changedAt)}`,
                meta: item.replacementRecommendation ? (
                  <StatusPill tone="accent">替代 {item.replacementRecommendation}</StatusPill>
                ) : null,
              }))}
            />
            {!governance.loading && !governance.errorDetail && governance.data?.items.length === 0 ? (
              <div className="empty-box">当前还没有治理事件记录。</div>
            ) : null}
          </SurfaceCard>
          <SurfaceCard description="审计事件用于确认哪些活动经过了注册表界面。" title="最新审计" variant="console">
            <TimelineList
              items={(audit.data?.items || []).map((item) => ({
                key: item.id,
                title: item.action,
                subtitle: `${item.actor} · ${item.statusCode} · ${formatDate(item.createdAt)}`,
                meta: <StatusPill tone={item.outcome === 'succeeded' ? 'good' : 'bad'}>{describeAuditOutcome(item.outcome)}</StatusPill>,
              }))}
            />
            {!audit.loading && !audit.errorDetail && audit.data?.items.length === 0 ? (
              <div className="empty-box">当前还没有审计事件记录。</div>
            ) : null}
          </SurfaceCard>
        </div>
      </div>
    </div>
  );
}

function describeOverviewError(error: RegistryRequestError) {
  if (error.kind === 'auth') {
    return `${error.message} 当前会话无法读取部分概览数据。`;
  }

  if (error.kind === 'network' || error.kind === 'server') {
    return `${error.message} 概览未能完整加载。`;
  }

  if (error.kind === 'not_found') {
    return `${error.message} 页面加载期间，某个概览数据集已发生变化。`;
  }

  return error.message;
}

function describeAuditOutcome(outcome: string) {
  return outcome === 'succeeded' ? '成功' : '失败';
}
