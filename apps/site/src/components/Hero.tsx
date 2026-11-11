import { ArrowRight, ArrowUpRight, Check, CircleCheck, Play } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { AGENTS, HERO_COMMAND, REPO_URL } from '../lib/content';
import { StatusPill } from './StatusPill';
import { AnimatedGridPattern } from './ui/AnimatedGridPattern';
import { BlurText } from './ui/BlurText';
import { BorderBeam } from './ui/BorderBeam';
import { Magnet, Tilt } from './ui/Magnet';
import { Terminal } from './ui/Terminal';

function ReleaseCard() {
  const [resolved, setResolved] = useState(false);

  return (
    <div className="release-card">
      <BorderBeam />
      <div className="release-card__top">
        <span className="release-card__label">catalog / release</span>
        <StatusPill>published</StatusPill>
      </div>

      <div className="release-card__title">
        <span className="release-card__package">@acme.skill/release-notes</span>
        <strong>1.4.0</strong>
      </div>

      <dl className="release-card__rows">
        <div>
          <dt>integrity</dt>
          <dd>sha512-9f2c…</dd>
        </div>
        <div>
          <dt>manifest</dt>
          <dd>sha256:6a1d…</dd>
        </div>
        <div>
          <dt>policy</dt>
          <dd>{resolved ? 'resolved' : 'resolvable'}</dd>
        </div>
      </dl>

      <div className="release-card__divider" />

      <div className="release-card__agents">
        <span className="release-card__meta-label">agents</span>
        <div className="agent-chips">
          {AGENTS.slice(0, 3).map((agent) => (
            <span className="agent-chip agent-chip--small" key={agent}>
              {agent}
            </span>
          ))}
          <span className="agent-chip agent-chip--small agent-chip--muted">+53</span>
        </div>
      </div>

      <button
        className="release-card__resolve"
        disabled={resolved}
        onClick={() => setResolved(true)}
        type="button"
      >
        {resolved ? (
          <>
            <Check aria-hidden="true" size={15} strokeWidth={2.2} />
            resolved in 38 ms
          </>
        ) : (
          <>
            <Play aria-hidden="true" size={14} strokeWidth={2.2} />
            resolve install
          </>
        )}
      </button>

      <p aria-live="polite" className="release-card__hint">
        {resolved ? (
          <>
            <CircleCheck aria-hidden="true" size={13} />
            Install target returned for 1.4.0
          </>
        ) : (
          'Example release record'
        )}
      </p>
    </div>
  );
}

export function Hero() {
  return (
    <section aria-labelledby="hero-title" className="hero">
      <div className="hero__copy">
        <p className="eyebrow">
          <span aria-hidden="true" className="eyebrow__dot" />
          Self-hosted · Open source
        </p>

        <h1 className="hero__title" id="hero-title">
          <BlurText text="Agent skills, with a release process." />
        </h1>

        <p className="hero__lede">
          Skill Dock is a self-hosted registry for agent skills. Bundles live in your npm registry.
          The catalog decides which release an agent gets and records every change to that decision.
        </p>

        <div className="hero__actions">
          <Magnet>
            <a className="button button--primary" href="#quick-start">
              Quick start
              <ArrowRight aria-hidden="true" size={16} strokeWidth={2} />
            </a>
          </Magnet>
          <a className="button button--secondary" href={REPO_URL} rel="noreferrer" target="_blank">
            View the source
            <ArrowUpRight aria-hidden="true" size={16} strokeWidth={2} />
          </a>
        </div>

        <Link className="hero__registry-link" to="/registry">
          Browse skills
          <ArrowRight aria-hidden="true" size={15} strokeWidth={2} />
        </Link>

        <div className="hero__command">
          <Terminal command={HERO_COMMAND} title="start the catalog" />
          <p className="hero__note">Starts the catalog, Postgres, and a local npm registry.</p>
        </div>
      </div>

      <div className="hero__visual">
        <AnimatedGridPattern className="hero__grid" />
        <div className="hero__visual-glow" aria-hidden="true" />
        <Tilt className="hero__tilt">
          <ReleaseCard />
        </Tilt>
      </div>
    </section>
  );
}
