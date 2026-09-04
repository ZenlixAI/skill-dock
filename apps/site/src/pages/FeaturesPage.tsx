import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';

const CONTROL_POINTS = [
  'Owns catalog truth and visibility.',
  'Decides what is installable and when.',
  'Tracks release records and governance.',
] as const;

const STORAGE_POINTS = [
  'Bundles live in a private npm-compatible repository.',
  'Storage stays external and replaceable.',
  'Storage carries bytes, not catalog truth.',
] as const;

const CLI_POINTS = [
  'Keeps discovery, install, publish, and import simple.',
  'Resolves registry-backed installs through the server first.',
  'Stays thin and predictable.',
] as const;

export function FeaturesPage() {
  return (
    <div className="page-grid page-grid--detail features-page">
      <section className="hero-shell hero-shell--split features-page__hero">
        <div className="hero-shell__copy features-page__copy">
          <SectionHeading
            title="Features"
            description="A self-hosted control plane for reusable skills."
            tone="frontstage"
          />

          <p className="features-page__lede">
            Keep discovery, governance, and bundle storage cleanly separated so the product is easy to run and easy to
            trust.
          </p>

          <div className="context-strip">
            <StatusPill tone="accent">Self-hosted</StatusPill>
            <StatusPill>Server authoritative</StatusPill>
            <StatusPill>Private npm-compatible storage</StatusPill>
          </div>
        </div>

        <SurfaceCard
          className="features-page__storage-card"
          description="Each layer has one job, and the product reads that way."
          title="Designed around clear boundaries"
          variant="hero"
        >
          <dl className="features-page__storage-list">
            <div>
              <dt>Control</dt>
              <dd>Server-side truth for visibility, lifecycle, and install decisions.</dd>
            </div>
            <div>
              <dt>Storage</dt>
              <dd>Bundle storage in a private npm-compatible artifact repository.</dd>
            </div>
            <div>
              <dt>CLI</dt>
              <dd>A thin command layer for discovery, install, publish, and import.</dd>
            </div>
          </dl>
        </SurfaceCard>
      </section>

      <section className="features-page__section">
        <div className="features-page__section-head">
          <h2>One product, three responsibilities</h2>
          <p>Control, storage, and command execution stay separate so the experience stays clear.</p>
        </div>

        <div className="features-page__matrix">
          <article className="features-page__column">
            <h3>Control</h3>
            <ul>
              {CONTROL_POINTS.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </article>
          <article className="features-page__column">
            <h3>Storage</h3>
            <ul>
              {STORAGE_POINTS.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </article>
          <article className="features-page__column">
            <h3>CLI</h3>
            <ul>
              {CLI_POINTS.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </article>
        </div>
      </section>

      <section className="features-page__closing">
        <p>Built to stay self-hosted, predictable, and easy to explain.</p>
      </section>
    </div>
  );
}
