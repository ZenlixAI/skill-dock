import { ArrowRight, ArrowUpRight, Check, CircleCheck, Play } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { AGENTS, HERO_COMMAND, REPO_URL } from '../lib/content';
import { useI18n } from '../lib/i18n';
import { StatusPill } from './StatusPill';
import { AnimatedGridPattern } from './ui/AnimatedGridPattern';
import { BlurText } from './ui/BlurText';
import { BorderBeam } from './ui/BorderBeam';
import { Magnet, Tilt } from './ui/Magnet';
import { Terminal } from './ui/Terminal';

function ReleaseCard() {
  const [resolved, setResolved] = useState(false);
  const { content } = useI18n();
  const card = content.hero.card;

  return (
    <div className="release-card">
      <BorderBeam />
      <div className="release-card__top">
        <span className="release-card__label">{card.catalogLabel}</span>
        <StatusPill>{card.published}</StatusPill>
      </div>

      <div className="release-card__title">
        <span className="release-card__package">@acme.skill/release-notes</span>
        <strong>1.4.0</strong>
      </div>

      <dl className="release-card__rows">
        <div>
          <dt>{card.integrity}</dt>
          <dd>sha512-9f2c…</dd>
        </div>
        <div>
          <dt>{card.manifest}</dt>
          <dd>sha256:6a1d…</dd>
        </div>
        <div>
          <dt>{card.policy}</dt>
          <dd>{resolved ? card.resolved : card.resolvable}</dd>
        </div>
      </dl>

      <div className="release-card__divider" />

      <div className="release-card__agents">
        <span className="release-card__meta-label">{card.agents}</span>
        <div className="agent-chips">
          {AGENTS.slice(0, 3).map((agent) => (
            <span className="agent-chip agent-chip--small" key={agent}>
              {agent}
            </span>
          ))}
          <span className="agent-chip agent-chip--small agent-chip--muted">{card.extraAgents}</span>
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
            {card.resolvedIn}
          </>
        ) : (
          <>
            <Play aria-hidden="true" size={14} strokeWidth={2.2} />
            {card.resolveInstall}
          </>
        )}
      </button>

      <p aria-live="polite" className="release-card__hint">
        {resolved ? (
          <>
            <CircleCheck aria-hidden="true" size={13} />
            {card.installTarget}
          </>
        ) : (
          card.example
        )}
      </p>
    </div>
  );
}

export function Hero() {
  const { content, path } = useI18n();
  const hero = content.hero;

  return (
    <section aria-labelledby="hero-title" className="hero">
      <div className="hero__copy">
        <p className="eyebrow">
          <span aria-hidden="true" className="eyebrow__dot" />
          {hero.eyebrow}
        </p>

        <h1 className="hero__title" id="hero-title">
          <BlurText text={hero.title} />
        </h1>

        <p className="hero__lede">{hero.lede}</p>

        <div className="hero__actions">
          <Magnet>
            <a className="button button--primary" href="#quick-start">
              {hero.quickStart}
              <ArrowRight aria-hidden="true" size={16} strokeWidth={2} />
            </a>
          </Magnet>
          <a className="button button--secondary" href={REPO_URL} rel="noreferrer" target="_blank">
            {hero.viewSource}
            <ArrowUpRight aria-hidden="true" size={16} strokeWidth={2} />
          </a>
        </div>

        <Link className="hero__registry-link" to={path('/registry')}>
          {hero.browseSkills}
          <ArrowRight aria-hidden="true" size={15} strokeWidth={2} />
        </Link>

        <div className="hero__command">
          <Terminal command={HERO_COMMAND} title={hero.terminalTitle} />
          <p className="hero__note">{hero.terminalNote}</p>
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
