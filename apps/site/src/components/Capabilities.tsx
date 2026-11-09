import { ArrowRight, Check, CircleSlash } from 'lucide-react';
import { CAPABILITIES } from '../lib/content';
import type { Capability } from '../lib/content';
import { SectionHeader } from './SectionHeader';
import { StatusPill } from './StatusPill';
import { Reveal } from './ui/BlurText';
import { SpotlightCard } from './ui/SpotlightCard';
import clsx from '../vendor/clsx';

function CapabilityVisual({ visual }: { visual: Capability['visual'] }) {
  if (visual === 'versions') {
    return (
      <div className="mini-list" aria-hidden="true">
        <div className="mini-row mini-row--active">
          <span className="mono">1.4.0</span>
          <StatusPill>published</StatusPill>
          <Check size={13} strokeWidth={2.2} />
        </div>
        <div className="mini-row">
          <span className="mono">1.3.1</span>
          <StatusPill>published</StatusPill>
        </div>
        <div className="mini-row mini-row--muted">
          <span className="mono">1.2.0</span>
          <StatusPill tone="amber">deprecated</StatusPill>
        </div>
      </div>
    );
  }

  if (visual === 'resolve') {
    return (
      <div className="mini-flow" aria-hidden="true">
        <span className="mini-chip">catalog</span>
        <ArrowRight size={12} />
        <span className="mini-chip mini-chip--accent">1.4.0</span>
        <ArrowRight size={12} />
        <span className="mini-chip">agent</span>
      </div>
    );
  }

  if (visual === 'policy') {
    return (
      <div className="mini-pills" aria-hidden="true">
        <StatusPill>active</StatusPill>
        <StatusPill tone="amber">deprecated</StatusPill>
        <StatusPill tone="red">yanked</StatusPill>
        <StatusPill tone="muted">
          <CircleSlash size={11} />
          blocked
        </StatusPill>
      </div>
    );
  }

  if (visual === 'audit') {
    return (
      <div className="mini-audit" aria-hidden="true">
        <div>
          <span className="mono">retire</span>
          <span className="mini-audit__meta">alice · 2d ago</span>
        </div>
        <p>reason: replaced by release-notes-v2</p>
      </div>
    );
  }

  if (visual === 'storage') {
    return (
      <div className="mini-pills" aria-hidden="true">
        <span className="mini-chip">Nexus</span>
        <span className="mini-chip">Artifactory</span>
        <span className="mini-chip">Verdaccio</span>
      </div>
    );
  }

  return (
    <div className="mini-delivery" aria-hidden="true">
      <div>
        <span className="mini-chip mini-chip--accent">proxy</span>
        <span className="mini-delivery__path">client → catalog → npm</span>
      </div>
      <div>
        <span className="mini-chip">direct</span>
        <span className="mini-delivery__path">client → npm</span>
      </div>
    </div>
  );
}

export function Capabilities() {
  return (
    <section aria-labelledby="capabilities-title" className="section capabilities">
      <Reveal>
        <SectionHeader
          eyebrow="What the catalog owns"
          id="capabilities-title"
          lede="Your npm registry already stores packages. The catalog adds the release decisions that packages do not carry."
          title="The catalog layer."
        />
      </Reveal>

      <Reveal delay={0.05}>
        <div className="capability-grid">
          {CAPABILITIES.map((capability, index) => {
            const Icon = capability.icon;
            return (
              <div
                className={clsx(
                  'capability-card',
                  index === 0 && 'capability-card--wide',
                  index === 5 && 'capability-card--full'
                )}
                key={capability.title}
              >
                <SpotlightCard className="capability-card__inner">
                  <div className="capability-card__head">
                    <span
                      className={`capability-card__icon capability-card__icon--${capability.accent}`}
                    >
                      <Icon aria-hidden="true" size={18} strokeWidth={1.8} />
                    </span>
                    <h3>{capability.title}</h3>
                  </div>
                  <p>{capability.body}</p>
                  <CapabilityVisual visual={capability.visual} />
                </SpotlightCard>
              </div>
            );
          })}
        </div>
      </Reveal>
    </section>
  );
}
