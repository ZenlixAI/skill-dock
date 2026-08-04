import { startTransition, useDeferredValue, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { listSkills } from '../api/client';
import type { CatalogSourceType, RegistryRequestError, SkillStatus } from '../api/types';
import { SectionHeading } from '../components/SectionHeading';
import { SkillSummaryCard } from '../components/SkillSummaryCard';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { usePaginatedCollection } from '../lib/usePaginatedCollection';

export function CatalogPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [query, setQuery] = useState(searchParams.get('q') ?? '');
  const [sourceType, setSourceType] = useState<CatalogSourceType | ''>('');
  const [status, setStatus] = useState<SkillStatus | ''>('');
  const deferredQuery = useDeferredValue(query);

  useEffect(() => {
    const nextQuery = searchParams.get('q') ?? '';
    setQuery((current) => (current === nextQuery ? current : nextQuery));
  }, [searchParams]);

  function updateQuery(next: string) {
    setQuery(next);

    startTransition(() => {
      const params = new URLSearchParams(searchParams);
      const normalized = next.trim();

      if (normalized) {
        params.set('q', normalized);
      } else {
        params.delete('q');
      }

      setSearchParams(params, { replace: true });
    });
  }

  const skills = usePaginatedCollection({
    loader: (filters) =>
      listSkills({
        ...filters,
        query: deferredQuery.trim(),
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
        description="按能力搜索，按来源或生命周期状态收窄结果，并在同一结果界面打开可直接安装的技能页面。"
        eyebrow="目录"
        title="发现注册表技能"
        tone="frontstage"
      />

      <SurfaceCard className="portal-hero" title="搜索" variant="hero">
        <div className="catalog-command">
          <label className="field catalog-command__search">
            <span>搜索技能</span>
            <input onChange={(event) => updateQuery(event.target.value)} placeholder="pdf / 评审 / 抓取 / 治理" value={query} />
          </label>
          <div className="catalog-command__status">
            <StatusPill tone="accent">查询 {deferredQuery || '全部技能'}</StatusPill>
            <StatusPill>{describeSourceType(sourceType)}</StatusPill>
            <StatusPill tone={status === 'retired' ? 'bad' : status === 'active' ? 'good' : undefined}>{describeSkillStatus(status)}</StatusPill>
            <StatusPill tone="good">当前 {resultItems.length}</StatusPill>
          </div>
        </div>
      </SurfaceCard>

      <div className="console-strip">
        <article className="console-signal">
          <span>查询范围</span>
          <strong>{deferredQuery ? '已过滤' : '开放'}</strong>
          <p>{deferredQuery ? '结果已按当前查询词收窄。' : '结果当前覆盖整个目录切片。'}</p>
        </article>
        <article className="console-signal">
          <span>启用 / 退役</span>
          <strong>{activeCount} / {retiredCount}</strong>
          <p>当前结果集的生命周期状态。</p>
        </article>
        <article className="console-signal">
          <span>导入来源</span>
          <strong>{importedCount}</strong>
          <p>当前结果中来自导入的技能数量。</p>
        </article>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard description="在按用途组织目录的同时，也可按安装成本代理信息和来源收窄范围。" title="筛选" variant="rail">
          <label className="field">
            <span>来源类型</span>
            <select value={sourceType} onChange={(event) => setSourceType(event.target.value as CatalogSourceType | '')}>
              <option value="">全部</option>
              <option value="native">原生</option>
              <option value="imported">导入</option>
              <option value="platform_created">平台创建</option>
            </select>
          </label>
          <label className="field">
            <span>状态</span>
            <select value={status} onChange={(event) => setStatus(event.target.value as SkillStatus | '')}>
              <option value="">全部</option>
              <option value="active">启用</option>
              <option value="retired">退役</option>
            </select>
          </label>
        </SurfaceCard>

        <SurfaceCard description="打开任一技能即可查看安装命令、版本状态和包级详情。" title="结果" variant="spotlight">
          {skills.loading && resultItems.length === 0 ? (
            <div aria-hidden="true" className="results-skeleton">
              <div className="portal-skeleton" />
              <div className="portal-skeleton" />
              <div className="portal-skeleton" />
            </div>
          ) : null}
          {skills.refreshing ? <div className="empty-box">正在刷新目录结果，并保留上一次成功加载的视图。</div> : null}
          {catalogErrorMessage ? <div className="error-box">{catalogErrorMessage}</div> : null}

          <div className="console-strip console-strip--compact">
            <article className="console-signal console-signal--compact">
              <span>结果</span>
              <strong>{resultItems.length}</strong>
            </article>
            <article className="console-signal console-signal--compact">
              <span>启用</span>
              <strong>{activeCount}</strong>
            </article>
            <article className="console-signal console-signal--compact">
              <span>退役</span>
              <strong>{retiredCount}</strong>
            </article>
          </div>

          <div className="list-stack">
            {resultItems.map((skill) => <SkillSummaryCard key={skill.packageName} skill={skill} />)}
            {!skills.loading && !skills.errorDetail && resultItems.length === 0 ? (
              <div className="empty-box">
                <strong>当前视图没有匹配技能。</strong>
                <p>请尝试更宽泛的能力词、包范围或来源筛选。</p>
              </div>
            ) : null}
          </div>

          {!skills.loading && !(skills.errorDetail && resultItems.length === 0) ? (
            <div className="action-row">
              <div className="action-row__copy">
                <strong>分页</strong>
                <p>{skills.pageInfo.hasNextPage ? `当前切片显示 ${resultItems.length} 条结果。` : `当前视图已显示全部 ${resultItems.length} 条结果。`}</p>
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
      ? `${error.message} 上一次成功加载的目录结果仍然可见。`
      : `${error.message} 当前会话暂时无法读取实时目录结果。`;
  }

  if (error.kind === 'network') {
    return hasStaleResults
      ? `${error.message} 刷新失败，因此仍显示上一次成功加载的结果。`
      : `${error.message} 实时后端不可用，因此目录结果未能加载。`;
  }

  if (error.kind === 'server') {
    return hasStaleResults
      ? `${error.message} 后端返回错误，因此仍显示上一次成功加载的结果。`
      : error.message;
  }

  if (error.kind === 'not_found') {
    return hasStaleResults
      ? `${error.message} 先保留之前的结果，直到后续刷新成功。`
      : error.message;
  }

  return error.message;
}

function describeSourceType(sourceType: CatalogSourceType | '') {
  if (sourceType === 'native') {
    return '原生';
  }

  if (sourceType === 'imported') {
    return '导入';
  }

  if (sourceType === 'platform_created') {
    return '平台创建';
  }

  return '全部来源';
}

function describeSkillStatus(status: SkillStatus | '') {
  if (status === 'active') {
    return '启用';
  }

  if (status === 'retired') {
    return '退役';
  }

  return '全部状态';
}
