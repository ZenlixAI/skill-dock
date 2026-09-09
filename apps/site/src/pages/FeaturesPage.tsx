import { RefreshCw } from 'lucide-react';
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
  'Keeps discovery, install, publish, import, and sync simple.',
  'Resolves registry-backed installs through the server first.',
  'Stays thin and predictable.',
] as const;

const SYNC_CONFIG = [
  '{',
  '  "syncOnAdd": true,',
  '  "packageNamePrefix": "@acme.skill/",',
  '  "defaultVersion": "0.1.0"',
  '}',
].join('\n');

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
            Keep discovery, governance, and bundle storage cleanly separated while giving source installs a direct path
            back to the backend.
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
              <dd>A thin command layer for discovery, install, publish, import, and automatic source sync.</dd>
            </div>
          </dl>
        </SurfaceCard>
      </section>

      <section className="features-page__section">
        <div className="features-page__section-head">
          <h2>One product, three responsibilities</h2>
          <p>Control, storage, and command execution stay separate; source installs still have a clear path back to the backend.</p>
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
        <div className="features-page__architecture" aria-label="Skill Registry architecture">
          <div><strong>CLI</strong><span>find · add · publish</span></div>
          <span className="features-page__architecture-arrow" aria-hidden="true">→</span>
          <div><strong>Registry API</strong><span>catalog truth · auth · governance</span></div>
          <span className="features-page__architecture-arrow" aria-hidden="true">↘</span>
          <div><strong>Private npm storage</strong><span>versioned skill bundles</span></div>
        </div>
      </section>

      <section className="features-page__section features-page__section--sync">
        <div className="features-page__section-head">
          <div className="features-page__sync-heading">
            <RefreshCw aria-hidden="true" size={18} strokeWidth={1.8} />
            <h2>Keep source installs connected</h2>
          </div>
          <p>
            Use <code>skills add</code> for local paths and git sources as usual. Successful installs are sent through
            the existing backend import flow without a second command.
          </p>
        </div>

        <div className="features-page__sync-grid">
          <ol className="features-page__sync-flow" aria-label="Automatic sync flow">
            <li>
              <span>1</span>
              <div>
                <strong>Install</strong>
                <p>Run <code>skills add ./my-skill</code> or install from a git source.</p>
              </div>
            </li>
            <li>
              <span>2</span>
              <div>
                <strong>Sync</strong>
                <p><code>syncOnAdd</code> sends the installed source through the existing backend workflow.</p>
              </div>
            </li>
            <li>
              <span>3</span>
              <div>
                <strong>Keep defaults consistent</strong>
                <p>Use a package prefix and version when the source does not supply package metadata.</p>
              </div>
            </li>
          </ol>

          <SurfaceCard
            className="features-page__sync-config"
            description="Set the control variable in the directory where you run the CLI."
            title="skills.config.json"
            variant="subtle"
          >
            <pre className="features-page__code-block">
              <code>{SYNC_CONFIG}</code>
            </pre>
            <p className="features-page__sync-note">
              Set <code>syncOnAdd</code> to <code>false</code> when an installation should remain local only.
            </p>
          </SurfaceCard>
        </div>
      </section>
    </div>
  );
}
