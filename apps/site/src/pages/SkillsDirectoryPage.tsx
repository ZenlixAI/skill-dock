import { ArrowRight, ArrowUpRight, RefreshCw, Search, X } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { ExternalSkillDrawer, formatInstalls } from '../components/ExternalSkillDrawer';
import type {
  ExternalSkill,
  ExternalSkillsPage,
  SkillsShBrowseView,
  SkillsShCapabilities,
} from '../lib/skills-directory-api';
import {
  fetchSkillsShBrowse,
  fetchSkillsShCapabilities,
  fetchSkillsShSearch,
  skillsShBrowseURL,
  skillsShSearchURL,
} from '../lib/skills-directory-api';
import clsx from '../vendor/clsx';

const PAGE_SIZE = 12;
const isPrerender = typeof window !== 'undefined' && new URLSearchParams(window.location.search).has('prerender');

const VIEWS: Array<{ value: SkillsShBrowseView; label: string }> = [
  { value: 'all-time', label: 'All-time' },
  { value: 'trending', label: 'Trending' },
  { value: 'hot', label: 'Hot' },
];

export function SkillsDirectoryPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [query, setQuery] = useState(() => searchParams.get('q') || '');
  const [debouncedQuery, setDebouncedQuery] = useState(() => (searchParams.get('q') || '').trim());
  const [view, setView] = useState<SkillsShBrowseView>(() => parseView(searchParams.get('view')));

  const [capabilities, setCapabilities] = useState<SkillsShCapabilities | null>(null);
  const [capabilitiesLoading, setCapabilitiesLoading] = useState(true);
  const [capabilitiesError, setCapabilitiesError] = useState<string | null>(null);

  const [page, setPage] = useState<ExternalSkillsPage | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [retryToken, setRetryToken] = useState(0);

  const [activeExternal, setActiveExternal] = useState<ExternalSkill | null>(null);
  const closeExternal = useCallback(() => setActiveExternal(null), []);
  const lastWrittenSearch = useRef('');

  const enabled = capabilities?.externalSearch.enabled === true;

  useEffect(() => {
    const controller = new AbortController();
    setCapabilitiesLoading(true);
    setCapabilitiesError(null);
    fetchSkillsShCapabilities(controller.signal)
      .then((next) => setCapabilities(next))
      .catch((cause: unknown) => {
        if (isAbortError(cause) || isPrerender) return;
        setCapabilitiesError(cause instanceof Error ? cause.message : 'Could not check the skills directory.');
      })
      .finally(() => {
        if (!controller.signal.aborted && !isPrerender) setCapabilitiesLoading(false);
      });
    return () => controller.abort();
  }, [retryToken]);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedQuery(query.trim()), 280);
    return () => window.clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    const params = new URLSearchParams();
    if (debouncedQuery) params.set('q', debouncedQuery);
    if (view !== 'all-time') params.set('view', view);
    lastWrittenSearch.current = params.toString();
    setSearchParams(params, { replace: true });
  }, [debouncedQuery, view, setSearchParams]);

  useEffect(() => {
    const current = searchParams.toString();
    if (current === lastWrittenSearch.current) return;
    setQuery(searchParams.get('q') || '');
    setView(parseView(searchParams.get('view')));
  }, [searchParams]);

  const browseKey = debouncedQuery ? '' : view;

  useEffect(() => {
    if (!enabled) return;
    if (debouncedQuery.length === 1) {
      setLoading(false);
      setError(null);
      setPage(null);
      return;
    }

    const controller = new AbortController();
    setLoading(true);
    setError(null);

    const request = debouncedQuery
      ? fetchSkillsShSearch({ query: debouncedQuery, pageSize: PAGE_SIZE }, controller.signal)
      : fetchSkillsShBrowse({ view, page: 1, pageSize: PAGE_SIZE }, controller.signal);

    request
      .then((next) => setPage(next))
      .catch((cause: unknown) => {
        if (isAbortError(cause)) return;
        setError(cause instanceof Error ? cause.message : 'The skills directory is unavailable.');
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });

    return () => controller.abort();
    // browseKey intentionally tracks the browse view without refetching search results when view changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [browseKey, debouncedQuery, enabled, retryToken]);

  useEffect(() => {
    if (!enabled && activeExternal) {
      setActiveExternal(null);
    }
  }, [enabled, activeExternal]);

  const retry = () => setRetryToken((value) => value + 1);
  const clearSearch = () => {
    setQuery('');
    setView('all-time');
  };

  const items = page?.items ?? [];
  const showViewTabs = enabled && !debouncedQuery;

  return (
    <div className="registry-page">
      <header className="registry-hero">
        <p className="eyebrow">
          <span aria-hidden="true" className="eyebrow__dot" />
          Registry · skills.sh
        </p>
        <h1>Skills Directory</h1>
        <p className="registry-hero__lede">
          Browse the public skills.sh directory. Install any skill from its source repository.
        </p>

        <div className="registry-search">
          <Search aria-hidden="true" size={18} strokeWidth={1.8} />
          <input
            aria-label="Search skills"
            disabled={!enabled}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search skills.sh"
            type="search"
            value={query}
          />
          {query ? (
            <button aria-label="Clear search" className="registry-search__clear" onClick={() => setQuery('')} type="button">
              <X aria-hidden="true" size={16} />
            </button>
          ) : null}
        </div>

        {enabled && query.trim().length === 1 ? (
          <p className="registry-search__hint">Type at least 2 characters to search skills.sh.</p>
        ) : null}

        {showViewTabs ? (
          <div aria-label="Browse view" className="registry-filters" role="group">
            {VIEWS.map((entry) => (
              <button
                aria-pressed={view === entry.value}
                className={clsx('registry-filter', view === entry.value && 'registry-filter--active')}
                key={entry.value}
                onClick={() => setView(entry.value)}
                type="button"
              >
                {entry.label}
              </button>
            ))}
          </div>
        ) : null}
      </header>

      {capabilitiesLoading ? (
        <CardSkeletonGrid />
      ) : capabilitiesError ? (
        <State
          action={retry}
          message={capabilitiesError}
          title="Could not check the skills directory"
        />
      ) : !enabled ? (
        <State
          message="This deployment has disabled the public skills directory."
          title="Skills directory is not enabled"
        />
      ) : debouncedQuery.length === 1 ? (
        <State
          message="Type at least 2 characters to search skills.sh."
          title="Keep typing"
        />
      ) : loading ? (
        <CardSkeletonGrid />
      ) : error ? (
        <State action={retry} message={error} title="Could not load the skills directory" />
      ) : items.length > 0 ? (
        <section aria-labelledby="skills-sh-results-title" className="registry-group registry-group--external">
          <div className="registry-group__header">
            <div>
              <h2 id="skills-sh-results-title">skills.sh</h2>
              <p>
                {debouncedQuery
                  ? `Search results for “${debouncedQuery}”.`
                  : `${viewDescription(view)}.`}{' '}
                A community directory, not reviewed or hosted by Skill Dock.
              </p>
            </div>
            <a
              className="registry-group__link"
              href={debouncedQuery ? skillsShSearchURL(debouncedQuery) : skillsShBrowseURL(view)}
              rel="noreferrer"
              target="_blank"
            >
              View all on skills.sh
              <span className="sr-only"> (opens in a new tab)</span>
              <ArrowUpRight aria-hidden="true" size={14} />
            </a>
          </div>

          <div className="registry-grid">
            {items.map((skill) => (
              <ExternalSkillCard key={skill.id} onOpen={() => setActiveExternal(skill)} skill={skill} />
            ))}
          </div>

          <div className="registry-more">
            {!debouncedQuery && page?.total && page.total > items.length ? (
              <span className="registry-more__hint">
                Showing {items.length} of {formatInstalls(page.total)}
              </span>
            ) : null}
          </div>
        </section>
      ) : (
        <State
          action={clearSearch}
          actionIcon="clear"
          actionLabel="Clear search"
          message={
            debouncedQuery
              ? `No skills.sh results for “${debouncedQuery}”.`
              : 'No skills in this view right now.'
          }
          title="Nothing found"
        />
      )}

      <ExternalSkillDrawer
        onClose={closeExternal}
        onSelectRelated={(skill) => setActiveExternal(skill)}
        open={Boolean(activeExternal)}
        skillId={activeExternal?.skillId ?? ''}
        source={activeExternal?.source ?? ''}
      />
    </div>
  );
}

function ExternalSkillCard({ skill, onOpen }: { skill: ExternalSkill; onOpen: () => void }) {
  return (
    <article className="registry-card registry-card--external">
      <div className="registry-card__top">
        <span className="registry-source registry-source--external">
          skills.sh
          <ArrowUpRight aria-hidden="true" size={12} />
        </span>
        {skill.isOfficial ? <span className="registry-official">Official</span> : null}
      </div>
      <h3>{skill.name}</h3>
      <p className="registry-card__package">{skill.source}</p>
      <div className="registry-card__meta">
        <span>{formatInstalls(skill.installs)} installs</span>
      </div>
      <div className="registry-card__actions">
        <button className="button button--secondary button--small" onClick={onOpen} type="button">
          View details
          <ArrowRight aria-hidden="true" size={14} />
        </button>
      </div>
    </article>
  );
}

function State({
  title,
  message,
  action,
  actionLabel = 'Retry',
  actionIcon = 'retry',
}: {
  title: string;
  message: string;
  action?: () => void;
  actionLabel?: string;
  actionIcon?: 'retry' | 'clear';
}) {
  return (
    <div className="registry-state registry-state--empty">
      <h2>{title}</h2>
      <p>{message}</p>
      {action ? (
        <button className="button button--secondary button--small" onClick={action} type="button">
          {actionIcon === 'clear' ? (
            <X aria-hidden="true" size={14} />
          ) : (
            <RefreshCw aria-hidden="true" size={14} />
          )}
          {actionLabel}
        </button>
      ) : null}
    </div>
  );
}

function CardSkeletonGrid() {
  return (
    <div aria-busy="true" className="registry-grid">
      {[0, 1, 2].map((index) => (
        <div className="registry-card registry-card--skeleton" key={index}>
          <span className="registry-skeleton registry-skeleton--short" />
          <span className="registry-skeleton registry-skeleton--title" />
          <span className="registry-skeleton" />
          <span className="registry-skeleton registry-skeleton--short" />
        </div>
      ))}
    </div>
  );
}

function parseView(raw: string | null): SkillsShBrowseView {
  if (raw === 'trending' || raw === 'hot') return raw;
  return 'all-time';
}

function viewDescription(view: SkillsShBrowseView): string {
  switch (view) {
    case 'trending':
      return 'Trending skills from the last 24 hours';
    case 'hot':
      return 'Skills with the biggest short-term movement';
    default:
      return 'Popular skills by all-time installs';
  }
}

function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === 'AbortError';
}
