import { startTransition, useDeferredValue, useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
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
        description="Search by capability, narrow by source or lifecycle status, and open install-ready skill pages from the same results surface."
        eyebrow="Catalog"
        title="Discover registry skills"
        tone="frontstage"
      />

      <SurfaceCard className="portal-hero" title="Search" variant="hero">
        <div className="catalog-command">
          <label className="field catalog-command__search">
            <span>Search skills</span>
            <input onChange={(event) => updateQuery(event.target.value)} placeholder="pdf / review / scrape / governance" value={query} />
          </label>
          <div className="catalog-command__status">
            <StatusPill tone="accent">Query {deferredQuery || 'all skills'}</StatusPill>
            <StatusPill>{sourceType || 'all sources'}</StatusPill>
            <StatusPill tone={status === 'retired' ? 'bad' : status === 'active' ? 'good' : undefined}>{status || 'all statuses'}</StatusPill>
            <StatusPill tone="good">Showing {resultItems.length}</StatusPill>
          </div>
        </div>
      </SurfaceCard>

      <div className="console-strip">
        <article className="console-signal">
          <span>Query scope</span>
          <strong>{deferredQuery ? 'Filtered' : 'Open'}</strong>
          <p>{deferredQuery ? 'Results are narrowed by the current query string.' : 'Results currently span the full catalog slice.'}</p>
        </article>
        <article className="console-signal">
          <span>Active / retired</span>
          <strong>{activeCount} / {retiredCount}</strong>
          <p>Lifecycle posture across the current result set.</p>
        </article>
        <article className="console-signal">
          <span>Imported</span>
          <strong>{importedCount}</strong>
          <p>Imported skills represented in the current results.</p>
        </article>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard description="Keep the catalog ordered around usefulness while narrowing by install cost proxies and provenance." title="Filters" variant="rail">
          <label className="field">
            <span>Source type</span>
            <select value={sourceType} onChange={(event) => setSourceType(event.target.value as CatalogSourceType | '')}>
              <option value="">All</option>
              <option value="native">native</option>
              <option value="imported">imported</option>
              <option value="platform_created">platform_created</option>
            </select>
          </label>
          <label className="field">
            <span>Status</span>
            <select value={status} onChange={(event) => setStatus(event.target.value as SkillStatus | '')}>
              <option value="">All</option>
              <option value="active">active</option>
              <option value="retired">retired</option>
            </select>
          </label>
        </SurfaceCard>

        <SurfaceCard description="Open any skill to inspect install commands, version posture, and package-level detail." title="Results" variant="spotlight">
          {skills.loading && resultItems.length === 0 ? (
            <div aria-hidden="true" className="results-skeleton">
              <div className="portal-skeleton" />
              <div className="portal-skeleton" />
              <div className="portal-skeleton" />
            </div>
          ) : null}
          {skills.refreshing ? <div className="empty-box">Refreshing catalog results while preserving the last successful view.</div> : null}
          {catalogErrorMessage ? <div className="error-box">{catalogErrorMessage}</div> : null}

          <div className="console-strip console-strip--compact">
            <article className="console-signal console-signal--compact">
              <span>Results</span>
              <strong>{resultItems.length}</strong>
            </article>
            <article className="console-signal console-signal--compact">
              <span>Active</span>
              <strong>{activeCount}</strong>
            </article>
            <article className="console-signal console-signal--compact">
              <span>Retired</span>
              <strong>{retiredCount}</strong>
            </article>
          </div>

          <div className="list-stack">
            {resultItems.map((skill) => <SkillSummaryCard key={skill.packageName} skill={skill} />)}
            {!skills.loading && !skills.errorDetail && resultItems.length === 0 ? (
              <div className="empty-box">
                <strong>No skills matched this view.</strong>
                <p>Try a broader capability, package scope, or source filter.</p>
              </div>
            ) : null}
          </div>

          {!skills.loading && !(skills.errorDetail && resultItems.length === 0) ? (
            <div className="action-row">
              <div className="action-row__copy">
                <strong>Pagination</strong>
                <p>{skills.pageInfo.hasNextPage ? `Showing ${resultItems.length} results in the current slice.` : `Showing all ${resultItems.length} results in this view.`}</p>
              </div>
              {skills.pageInfo.hasNextPage ? (
                <button className="button button--secondary" disabled={skills.loadingMore} onClick={skills.loadMore} type="button">
                  {skills.loadingMore ? 'Loading…' : 'Load more'}
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
      ? `${error.message} The last successful catalog results are still visible.`
      : `${error.message} This session cannot read live catalog results right now.`;
  }

  if (error.kind === 'network') {
    return hasStaleResults
      ? `${error.message} Refresh failed, so the last successful results are still shown.`
      : `${error.message} The live backend is unavailable, so catalog results could not load.`;
  }

  if (error.kind === 'server') {
    return hasStaleResults
      ? `${error.message} The backend returned an error, so the last successful results are still shown.`
      : error.message;
  }

  if (error.kind === 'not_found') {
    return hasStaleResults
      ? `${error.message} Previous results are preserved until a later refresh succeeds.`
      : error.message;
  }

  return error.message;
}
