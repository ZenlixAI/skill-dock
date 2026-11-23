import { ArrowRight, ArrowUpRight, RefreshCw, Search, Star, X } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { ExternalSkillDrawer } from '../components/ExternalSkillDrawer';
import { externalRefKey, favoriteFromExternal, useFavorites } from '../lib/favorites';
import { formatInstalls } from '../lib/format';
import { useI18n } from '../lib/i18n';
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

export function SkillsDirectoryPage() {
  const { content, locale } = useI18n();
  const copy = content.registry;
  const favorites = useFavorites();
  const [savedOnly, setSavedOnly] = useState(false);
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
        setCapabilitiesError(
          cause instanceof Error ? cause.message : copy.capabilityCheckErrorFallback
        );
      })
      .finally(() => {
        if (!controller.signal.aborted && !isPrerender) setCapabilitiesLoading(false);
      });
    return () => controller.abort();
    // The localized fallback is read at request time; locale changes do not need a refetch.
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
        setError(cause instanceof Error ? cause.message : copy.loadErrorFallback);
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
  const savedItems: ExternalSkill[] = favorites.favorites
    .filter((favorite) => favorite.refType === 'external' && favorite.snapshot.name && favorite.externalSource && favorite.externalSkillId)
    .filter((favorite) => {
      const needle = debouncedQuery.toLowerCase();
      if (!needle) return true;
      return (
        (favorite.snapshot.name ?? '').toLowerCase().includes(needle) ||
        (favorite.snapshot.source ?? favorite.externalSource ?? '').toLowerCase().includes(needle)
      );
    })
    .map((favorite) => ({
      id: `${favorite.externalSource}/${favorite.externalSkillId}`,
      source: favorite.externalSource ?? '',
      skillId: favorite.externalSkillId ?? '',
      name: favorite.snapshot.name ?? '',
      installs: favorite.snapshot.installs ?? 0,
      isOfficial: favorite.snapshot.isOfficial,
      installCommand: favorite.snapshot.installCommand,
    }));

  return (
    <div className="registry-page">
      <header className="registry-hero">
        <p className="eyebrow">
          <span aria-hidden="true" className="eyebrow__dot" />
          {copy.eyebrow}
        </p>
        <h1>{copy.title}</h1>
        <p className="registry-hero__lede">{copy.lede}</p>

        <div className="registry-search">
          <Search aria-hidden="true" size={18} strokeWidth={1.8} />
          <input
            aria-label={copy.searchLabel}
            disabled={!enabled}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={copy.searchPlaceholder}
            type="search"
            value={query}
          />
          {query ? (
            <button
              aria-label={copy.clearSearch}
              className="registry-search__clear"
              onClick={() => setQuery('')}
              type="button"
            >
              <X aria-hidden="true" size={16} />
            </button>
          ) : null}
        </div>

        {enabled && query.trim().length === 1 ? (
          <p className="registry-search__hint">{copy.minCharsHint}</p>
        ) : null}

        {showViewTabs ? (
          <div aria-label={copy.viewsLabel} className="registry-filters" role="group">
            {copy.views.map((entry) => (
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
            <button
              aria-pressed={savedOnly}
              className={clsx('registry-filter registry-filter--saved', savedOnly && 'registry-filter--active')}
              onClick={() => setSavedOnly((current) => !current)}
              type="button"
            >
              <Star aria-hidden="true" fill={savedOnly ? 'currentColor' : 'none'} size={14} />
              {copy.savedCount(favorites.count)}
            </button>
          </div>
        ) : null}
      </header>

      {savedOnly ? (
        <section aria-labelledby="saved-skills-title" className="registry-group">
          <div className="registry-group__header">
            <div>
              <h2 id="saved-skills-title">{copy.savedTitle}</h2>
              <p>{copy.savedLocalHint}</p>
            </div>
          </div>
          {savedItems.length > 0 ? (
            <div className="registry-grid">
              {savedItems.map((skill) => (
                <ExternalSkillCard key={skill.id} onOpen={() => setActiveExternal(skill)} skill={skill} />
              ))}
            </div>
          ) : (
            <State message={copy.savedEmptyMessage} title={copy.savedEmptyTitle} />
          )}
        </section>
      ) : capabilitiesLoading ? (
        <CardSkeletonGrid />
      ) : capabilitiesError ? (
        <State
          action={retry}
          actionLabel={content.ui.retry}
          message={capabilitiesError}
          title={copy.capabilityCheckErrorTitle}
        />
      ) : !enabled ? (
        <State message={copy.disabledMessage} title={copy.disabledTitle} />
      ) : debouncedQuery.length === 1 ? (
        <State message={copy.minCharsHint} title={copy.keepTypingTitle} />
      ) : loading ? (
        <CardSkeletonGrid />
      ) : error ? (
        <State
          action={retry}
          actionLabel={content.ui.retry}
          message={error}
          title={copy.loadErrorTitle}
        />
      ) : items.length > 0 ? (
        <section aria-labelledby="skills-sh-results-title" className="registry-group registry-group--external">
          <div className="registry-group__header">
            <div>
              <h2 id="skills-sh-results-title">{copy.resultsTitle}</h2>
              <p>
                {debouncedQuery
                  ? copy.searchResultsFor(debouncedQuery)
                  : `${copy.viewDescriptions[view]}.`}{' '}
                {copy.communityNotice}
              </p>
            </div>
            <a
              className="registry-group__link"
              href={debouncedQuery ? skillsShSearchURL(debouncedQuery) : skillsShBrowseURL(view)}
              rel="noreferrer"
              target="_blank"
            >
              {copy.viewAll}
              <span className="sr-only">{content.ui.opensInNewTab}</span>
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
                {copy.showing(String(items.length), formatInstalls(page.total, locale))}
              </span>
            ) : null}
          </div>
        </section>
      ) : (
        <State
          action={clearSearch}
          actionIcon="clear"
          actionLabel={copy.clearSearch}
          message={
            debouncedQuery ? copy.emptySearch(debouncedQuery) : copy.emptyView
          }
          title={copy.emptyTitle}
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
  const { content, locale } = useI18n();
  const copy = content.registry;
  const favorites = useFavorites();
  const refKey = externalRefKey(skill);
  const saved = favorites.isFavorite(refKey);

  return (
    <article className="registry-card registry-card--external">
      <div className="registry-card__top">
        <span className="registry-source registry-source--external">
          skills.sh
          <ArrowUpRight aria-hidden="true" size={12} />
        </span>
        <div className="registry-card__top-actions">
          {skill.isOfficial ? <span className="registry-official">{copy.official}</span> : null}
          <button
            aria-label={saved ? copy.removeSavedSkill : copy.saveSkill}
            aria-pressed={saved}
            className={clsx('registry-save', saved && 'registry-save--active')}
            onClick={(event) => {
              event.stopPropagation();
              void favorites.toggle(favoriteFromExternal(skill));
            }}
            type="button"
          >
            <Star aria-hidden="true" fill={saved ? 'currentColor' : 'none'} size={16} />
          </button>
        </div>
      </div>
      <h3>{skill.name}</h3>
      <p className="registry-card__package">{skill.source}</p>
      <div className="registry-card__meta">
        <span>{copy.installs(formatInstalls(skill.installs, locale))}</span>
      </div>
      <div className="registry-card__actions">
        <button className="button button--secondary button--small" onClick={onOpen} type="button">
          {copy.viewDetails}
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
  actionLabel,
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

function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === 'AbortError';
}
