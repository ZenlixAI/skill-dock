import { useDeferredValue, useState } from 'react';
import { Link } from 'react-router-dom';
import { listSkills } from '../api/client';
import type { CatalogSourceType, RegistryRequestError, SkillStatus } from '../api/types';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { usePaginatedCollection } from '../lib/usePaginatedCollection';

export function CatalogPage() {
  const [query, setQuery] = useState('');
  const [sourceType, setSourceType] = useState<CatalogSourceType | ''>('');
  const [status, setStatus] = useState<SkillStatus | ''>('');
  const deferredQuery = useDeferredValue(query);

  const skills = usePaginatedCollection({
    loader: (filters) =>
      listSkills({
        ...filters,
        query: deferredQuery,
        sourceType,
        status,
        pageSize: 12,
      }),
    filters: {
      cursor: undefined,
    },
    deps: [deferredQuery, sourceType, status],
    preserveStaleItemsOnRefresh: true,
  });
  const resultItems = skills.items;
  const activeCount = resultItems.filter((item) => item.status === 'active').length;
  const retiredCount = resultItems.filter((item) => item.status === 'retired').length;
  const importedCount = resultItems.filter((item) => item.sourceType === 'imported').length;
  const catalogErrorMessage = skills.errorDetail ? describeCatalogError(skills.errorDetail, resultItems.length > 0) : null;

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="目录"
        title="查找技能"
        tone="frontstage"
      />

      <SurfaceCard title="搜索" variant="hero">
        <div className="catalog-command">
          <label className="field catalog-command__search">
            <span>搜索</span>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="pdf / review / scrape / governance" />
          </label>
          <div className="catalog-command__status">
            <StatusPill tone="accent">查询 {deferredQuery || '全部'}</StatusPill>
            <StatusPill>{sourceType || '全部来源'}</StatusPill>
            <StatusPill tone={status === 'retired' ? 'bad' : status === 'active' ? 'good' : undefined}>{status || '全部状态'}</StatusPill>
            <StatusPill tone="good">已显示 {resultItems.length}</StatusPill>
          </div>
        </div>
      </SurfaceCard>

      <div className="console-strip">
        <article className="console-signal">
          <span>查询</span>
          <strong>{deferredQuery ? '已收窄' : '全量'}</strong>
          <p>{deferredQuery ? '当前正在展示过滤后的结果。' : '当前展示全部结果。'}</p>
        </article>
        <article className="console-signal">
          <span>活跃 / 退役</span>
          <strong>{activeCount} / {retiredCount}</strong>
          <p>当前生命周期分布。</p>
        </article>
        <article className="console-signal">
          <span>导入来源</span>
          <strong>{importedCount}</strong>
          <p>当前结果中的导入技能数量。</p>
        </article>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard title="筛选" variant="rail">
          <label className="field">
            <span>来源类型</span>
            <select value={sourceType} onChange={(event) => setSourceType(event.target.value as CatalogSourceType | '')}>
              <option value="">全部</option>
              <option value="native">native</option>
              <option value="imported">imported</option>
              <option value="platform_created">platform_created</option>
            </select>
          </label>
          <label className="field">
            <span>状态</span>
            <select value={status} onChange={(event) => setStatus(event.target.value as SkillStatus | '')}>
              <option value="">全部</option>
              <option value="active">active</option>
              <option value="retired">retired</option>
            </select>
          </label>
        </SurfaceCard>

        <SurfaceCard title="结果" variant="spotlight">
          {skills.loading && resultItems.length === 0 ? <div className="empty-box">正在加载目录…</div> : null}
          {skills.refreshing ? <div className="empty-box">正在刷新目录结果，同时保留上一次成功结果…</div> : null}
          {catalogErrorMessage ? <div className="error-box">{catalogErrorMessage}</div> : null}

          <div className="console-strip console-strip--compact">
            <article className="console-signal console-signal--compact">
              <span>结果</span>
              <strong>{resultItems.length}</strong>
            </article>
            <article className="console-signal console-signal--compact">
              <span>活跃</span>
              <strong>{activeCount}</strong>
            </article>
            <article className="console-signal console-signal--compact">
              <span>退役</span>
              <strong>{retiredCount}</strong>
            </article>
          </div>

          <div className="list-stack">
            {resultItems.map((skill) => (
              <article className="list-card list-card--feature list-card--spotlight" key={skill.packageName}>
                <div className="list-card__head">
                  <div>
                    <strong>{skill.displayName}</strong>
                    <span>{skill.packageName}</span>
                  </div>
                  <StatusPill tone={skill.latestVersionStatus === 'published' ? 'good' : skill.latestVersionStatus === 'deprecated' ? 'warn' : 'bad'}>
                    {skill.latestVersionStatus || 'unknown'}
                  </StatusPill>
                </div>
                <p>{skill.description}</p>
                <div className="pill-row">
                  <StatusPill tone="accent">{skill.sourceType}</StatusPill>
                  <StatusPill tone={skill.status === 'active' ? 'good' : 'bad'}>{skill.status}</StatusPill>
                  <StatusPill>latest {skill.latestVersion || '—'}</StatusPill>
                </div>
                <div className="list-card__actions">
                  <Link className="button button--secondary" to={`/skills/${encodeURIComponent(skill.packageName)}`}>
                    打开
                  </Link>
                  {skill.latestVersion ? (
                    <Link
                      className="button button--secondary"
                      to={`/skills/${encodeURIComponent(skill.packageName)}/versions/${skill.latestVersion}`}
                    >
                      最新版本
                    </Link>
                  ) : null}
                </div>
              </article>
            ))}
            {!skills.loading && !skills.errorDetail && resultItems.length === 0 ? (
              <div className="empty-box">当前筛选条件下没有匹配的技能。</div>
            ) : null}
          </div>

          {!skills.loading && !(skills.errorDetail && resultItems.length === 0) ? (
            <div className="action-row">
              <div className="action-row__copy">
                <strong>分页</strong>
                <p>{skills.pageInfo.hasNextPage ? `当前显示 ${resultItems.length} 条结果。` : `当前共有 ${resultItems.length} 条结果。`}</p>
              </div>
              {skills.pageInfo.hasNextPage ? (
                <button className="button button--secondary" disabled={skills.loadingMore} onClick={skills.loadMore} type="button">
                  {skills.loadingMore ? '加载中…' : '加载更多'}
                </button>
              ) : null}
            </div>
          ) : null}
        </SurfaceCard>
      </div>
    </div>
  );
}

function describeCatalogError(error: RegistryRequestError, hasStaleResults: boolean) {
  if (error.kind === 'auth') {
    return hasStaleResults
      ? `${error.message} 当前保留上一次成功的目录结果。`
      : `${error.message} 当前会话无法读取真实后端的目录结果。`;
  }

  if (error.kind === 'network') {
    return hasStaleResults
      ? `${error.message} 刷新失败，当前保留上一次成功的目录页。`
      : `${error.message} 真实后端不可用，目录结果未能加载。`;
  }

  if (error.kind === 'server') {
    return hasStaleResults
      ? `${error.message} 后端返回错误，当前保留上一次成功的目录页。`
      : error.message;
  }

  if (error.kind === 'not_found') {
    return hasStaleResults
      ? `${error.message} 下一次刷新成功前，当前保留之前的结果。`
      : error.message;
  }

  return error.message;
}
