import { ArrowRight, Check, CircleSlash } from 'lucide-react';
import type { SiteContent } from '../lib/content';
import { useI18n } from '../lib/i18n';
import { SectionHeader } from './SectionHeader';
import { StatusPill } from './StatusPill';
import { Reveal } from './ui/BlurText';
import { SpotlightCard } from './ui/SpotlightCard';
import clsx from '../vendor/clsx';

type Capability = SiteContent['capabilities']['items'][number];
type CapabilityVisualKind = Capability['visual'];

function CapabilityVisual({
  visual,
  labels,
}: {
  visual: CapabilityVisualKind;
  labels: SiteContent['capabilities']['visuals'];
}) {
  if (visual === 'versions') {
    return (
      <div className="mini-list" aria-hidden="true">
        <div className="mini-row mini-row--active">
          <span className="mono">1.4.0</span>
          <StatusPill>{labels.published}</StatusPill>
          <Check size={13} strokeWidth={2.2} />
        </div>
        <div className="mini-row">
          <span className="mono">1.3.1</span>
          <StatusPill>{labels.published}</StatusPill>
        </div>
        <div className="mini-row mini-row--muted">
          <span className="mono">1.2.0</span>
          <StatusPill tone="amber">{labels.deprecated}</StatusPill>
        </div>
      </div>
    );
  }

  if (visual === 'resolve') {
    return (
      <div className="mini-flow" aria-hidden="true">
        <span className="mini-chip">{labels.catalog}</span>
        <ArrowRight size={12} />
        <span className="mini-chip mini-chip--accent">1.4.0</span>
        <ArrowRight size={12} />
        <span className="mini-chip">{labels.agent}</span>
      </div>
    );
  }

  if (visual === 'policy') {
    return (
      <div className="mini-pills" aria-hidden="true">
        <StatusPill>{labels.active}</StatusPill>
        <StatusPill tone="amber">{labels.deprecated}</StatusPill>
        <StatusPill tone="red">{labels.yanked}</StatusPill>
        <StatusPill tone="muted">
          <CircleSlash size={11} />
          {labels.blocked}
        </StatusPill>
      </div>
    );
  }

  if (visual === 'audit') {
    return (
      <div className="mini-audit" aria-hidden="true">
        <div>
          <span className="mono">{labels.retire}</span>
          <span className="mini-audit__meta">{labels.auditMeta}</span>
        </div>
        <p>{labels.auditReason}</p>
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
        <span className="mini-chip mini-chip--accent">{labels.proxy}</span>
        <span className="mini-delivery__path">client → catalog → npm</span>
      </div>
      <div>
        <span className="mini-chip">{labels.direct}</span>
        <span className="mini-delivery__path">client → npm</span>
      </div>
    </div>
  );
}

export function Capabilities() {
  const { content } = useI18n();
  const copy = content.capabilities;

  return (
    <section aria-labelledby="capabilities-title" className="section capabilities">
      <Reveal>
        <SectionHeader
          eyebrow={copy.eyebrow}
          id="capabilities-title"
          lede={copy.lede}
          title={copy.title}
        />
      </Reveal>

      <Reveal delay={0.05}>
        <div className="capability-grid">
          {copy.items.map((capability, index) => {
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
                  <CapabilityVisual labels={copy.visuals} visual={capability.visual} />
                </SpotlightCard>
              </div>
            );
          })}
        </div>
      </Reveal>
    </section>
  );
}
