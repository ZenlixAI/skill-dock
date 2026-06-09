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
  const recentSkills = skills.data?.items.slice(0, 3) || [];
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
            <StatusPill tone="accent">Portal</StatusPill>
            <Link className="button button--primary" to="/catalog">
              Browse catalog
            </Link>
          </div>
        }
        description="Use the portal to search for capabilities, evaluate package provenance, and jump into install-ready detail pages."
        eyebrow="Overview"
        title="Start discovery from the skill registry."
        tone="frontstage"
      />

      <SurfaceCard className="portal-hero" variant="hero">
        <div className="hero-kicker">Discovery portal</div>
        <h2 className="hero-title">Search by capability first, then inspect install posture and package provenance.</h2>
        <p className="hero-copy">
          Recommended skills stay close at hand, while recent updates and console workflows remain available without turning the
          landing page into an operations dashboard.
        </p>

        <form className="portal-hero__search" onSubmit={handleSearchSubmit}>
          <label className="field">
            <span>Search skills</span>
            <input
              onChange={(event) => setQuery(event.target.value)}
              placeholder="review, pdf, governance, import"
              value={query}
            />
          </label>
          <button className="button button--primary" type="submit">
            Search catalog
          </button>
        </form>

        <div className="hero-band">
          <article className="hero-band__item">
            <span>Catalog slice</span>
            <strong>{skills.loading ? '—' : shownSkills}</strong>
            <p>{skills.data?.pageInfo.hasNextPage ? 'Showing the first discovery slice.' : 'Showing the current discovery slice.'}</p>
          </article>
          <article className="hero-band__item">
            <span>Recommended</span>
            <strong>{curated.loading ? '—' : curated.data?.length || 0}</strong>
            <p>Curated skills that are useful for common internal workflows.</p>
          </article>
          <article className="hero-band__item">
            <span>Console handoff</span>
            <strong>{governance.data?.items.length || audit.data?.items.length ? 'Live' : 'Ready'}</strong>
            <p>Governance and audit remain one step away when discovery needs deeper operational context.</p>
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
              Explore all skills
            </Link>
          }
          description="Recommended entries emphasize usefulness up front while keeping install-ready metadata close."
          title="Recommended skills"
          variant="spotlight"
        >
          <div className="list-stack">
            {curated.data?.map((skill) => (
              <SkillSummaryCard curationReason={skill.curationReason} key={skill.packageName} skill={skill} />
            ))}
            {!curated.loading && !curated.errorDetail && curated.data?.length === 0 ? (
              <div className="empty-box">
                <strong>No recommended skills are published yet.</strong>
                <p>Use the catalog to explore the full registry while curation catches up.</p>
              </div>
            ) : null}
          </div>
        </SurfaceCard>

        <div className="stack-column">
          <SurfaceCard className="console-entry-card" description="Jump into import, governance, and audit workflows when discovery turns into delivery." title="Console" variant="subtle">
            <div className="console-entry-card__actions">
              <Link className="button button--secondary" to="/workbench">
                Open console
              </Link>
              <Link className="button button--secondary" to="/workbench/audit">
                Review audit
              </Link>
            </div>
          </SurfaceCard>

          <SurfaceCard
            action={
              <Link className="button button--secondary" to="/catalog">
                See results
              </Link>
            }
            description="Recently updated skills keep discovery fresh without forcing operators into the console first."
            title="Recently updated"
            variant="spotlight"
          >
            <div className="list-stack">
              {recentSkills.map((skill) => (
                <SkillSummaryCard key={skill.packageName} skill={skill} />
              ))}
              {!skills.loading && !skills.errorDetail && recentSkills.length === 0 ? (
                <div className="empty-box">
                  <strong>No recent updates are available.</strong>
                  <p>Try the catalog for a broader browse or return later when new releases land.</p>
                </div>
              ) : null}
            </div>
          </SurfaceCard>

          <SurfaceCard description="Recent governance changes help explain why a package may have shifted posture." title="Latest governance" variant="console">
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
              <div className="empty-box">No governance events have been recorded yet.</div>
            ) : null}
          </SurfaceCard>
          <SurfaceCard description="Audit events confirm what activity has moved through the registry surface." title="Latest audit" variant="console">
            <TimelineList
              items={(audit.data?.items || []).map((item) => ({
                key: item.id,
                title: item.action,
                subtitle: `${item.actor} · ${item.statusCode} · ${formatDate(item.createdAt)}`,
                meta: <StatusPill tone={item.outcome === 'succeeded' ? 'good' : 'bad'}>{item.outcome}</StatusPill>,
              }))}
            />
            {!audit.loading && !audit.errorDetail && audit.data?.items.length === 0 ? (
              <div className="empty-box">No audit events have been recorded yet.</div>
            ) : null}
          </SurfaceCard>
        </div>
      </div>
    </div>
  );
}

function describeOverviewError(error: RegistryRequestError) {
  if (error.kind === 'auth') {
    return `${error.message} This session cannot read part of the overview data.`;
  }

  if (error.kind === 'network' || error.kind === 'server') {
    return `${error.message} The overview could not load completely.`;
  }

  if (error.kind === 'not_found') {
    return `${error.message} One of the overview datasets changed while the page was loading.`;
  }

  return error.message;
}
