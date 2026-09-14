import { useState } from 'react';
import { Copy, Package, Search, ShieldCheck } from 'lucide-react';
import { Link } from 'react-router-dom';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';

const SEARCH_COMMAND = 'npx skills@latest find <query>';

const FEATURE_BANDS = [
  {
    icon: Search,
    label: 'Find',
    title: 'Search the library your team already trusts.',
    description: 'Move from need to package without teaching people a new path.',
  },
  {
    icon: Package,
    label: 'Install',
    title: 'Use `skills add` for registry and source installs.',
    description: 'Automatic source sync with one control: keep installs connected to the backend without another command.',
  },
  {
    icon: ShieldCheck,
    label: 'Control',
    title: 'Keep governance close, not in the way.',
    description: 'Apply lifecycle and access rules without turning the homepage into a console.',
  },
] as const;

export function OverviewPage() {
  const [copyState, setCopyState] = useState<'idle' | 'copied' | 'failed'>('idle');

  async function handleCopyCommand() {
    try {
      await navigator.clipboard.writeText(SEARCH_COMMAND);
      setCopyState('copied');
    } catch {
      setCopyState('failed');
    }
  }

  const copyLabel = copyState === 'copied' ? 'Copied' : copyState === 'failed' ? 'Copy failed' : 'Copy command';

  return (
    <div className="page-grid page-grid--landing">
      <section className="hero-shell hero-shell--landing home-launch-shell">
        <div className="hero-shell__copy hero-shell__copy--landing">
          <div className="hero-kicker-row">
            <StatusPill tone="accent">Private catalog</StatusPill>
            <StatusPill>Versioned releases</StatusPill>
            <StatusPill>Auditable installs</StatusPill>
          </div>

          <h1 className="hero-title hero-title--landing">Private skills for every agent.</h1>

          <p className="hero-description">
            Discover, publish, and install governed agent skills from one catalog. Keep package identity, release decisions, and installation resolution under your team&apos;s control.
          </p>

          <div className="hero-actions">
            <Link className="button button--primary" to="/features">
              See how it works
            </Link>
            <a className="button button--ghost" href="https://github.com/ZenlixAI/skill-dock#docker-compose" rel="noreferrer" target="_blank">
              Install the registry
            </a>
            <span className="hero-actions__note">
              Start with <code>npx skills@latest</code>, then point it at your catalog.
            </span>
          </div>
        </div>

        <SurfaceCard
          action={
            <button
              aria-label={copyLabel}
              className="button button--ghost hero-command-copy"
              title={copyLabel}
              type="button"
              onClick={() => void handleCopyCommand()}
            >
              <Copy aria-hidden="true" size={14} strokeWidth={2} />
            </button>
            }
          className="hero-shell__panel hero-shell__panel--landing home-command-card"
          description="Search, install, and publish from one CLI. The catalog keeps releases predictable and governed."
          title="Start with a command"
          variant="hero"
        >
          <span className="sr-only" aria-live="polite">{copyState === 'copied' ? 'Command copied to clipboard.' : copyState === 'failed' ? 'Copy failed. Select the command manually.' : ''}</span>
          <div className="home-command-card__command">
            <code>{SEARCH_COMMAND}</code>
          </div>
          <div className="home-command-card__meta" aria-label="Command highlights">
            <span>Exact versions</span>
            <span>Verified artifacts</span>
            <span>Source sync</span>
          </div>
        </SurfaceCard>

        <section className="home-feature-bands" aria-label="Product highlights">
          {FEATURE_BANDS.map((band) => {
            const Icon = band.icon;

            return (
              <article className="home-feature-band" key={band.label}>
                <div className="home-feature-band__icon">
                  <Icon aria-hidden="true" size={16} strokeWidth={2} />
                </div>
                <div className="home-feature-band__copy">
                  <p>{band.label}</p>
                  <h2>{band.title}</h2>
                  <span>{band.description}</span>
                </div>
              </article>
            );
          })}
          <div className="home-feature-bands__callout">
            <StatusPill tone="accent">Optional source sync</StatusPill>
            <p>
              Local and Git installs can sync a copy to the catalog with <code>syncOnAdd</code>. Keep this advanced setting in your project configuration when you need it.
            </p>
          </div>
        </section>
      </section>
    </div>
  );
}
