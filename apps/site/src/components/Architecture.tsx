import { Boxes, Database, Server, TerminalSquare } from 'lucide-react';
import { useState } from 'react';
import { ARCHITECTURE_NODES } from '../lib/content';
import { SectionHeader } from './SectionHeader';
import { Reveal } from './ui/BlurText';
import { BorderBeam } from './ui/BorderBeam';

const FLOW = [
  { ...ARCHITECTURE_NODES[0], icon: TerminalSquare },
  { ...ARCHITECTURE_NODES[1], icon: Server },
  { ...ARCHITECTURE_NODES[3], icon: Boxes },
] as const;

export function Architecture() {
  const [active, setActive] = useState('catalog');
  const selected = ARCHITECTURE_NODES.find((node) => node.id === active) ?? ARCHITECTURE_NODES[1];

  return (
    <section aria-labelledby="architecture-title" className="section architecture" id="architecture">
      <Reveal>
        <SectionHeader
          eyebrow="Architecture"
          id="architecture-title"
          lede="The catalog is a service you run. Postgres holds release records and governance data. Your npm registry holds the bundle bytes. The CLI runs on a developer machine or in CI."
          title="Three pieces, one release path."
        />
      </Reveal>

      <Reveal delay={0.05}>
        <div className="arch-shell">
          <BorderBeam duration={12} size={180} />
          <div aria-label="Architecture flow" className="arch-flow" role="group">
            {FLOW.map((node, index) => {
              const Icon = node.icon;
              const isActive = active === node.id;
              return (
                <div className="arch-flow__item" key={node.id}>
                  <button
                    aria-pressed={isActive}
                    className={`arch-node${isActive ? ' arch-node--active' : ''}`}
                    onClick={() => setActive(node.id)}
                    type="button"
                  >
                    <span className="arch-node__icon">
                      <Icon aria-hidden="true" size={19} strokeWidth={1.7} />
                    </span>
                    <span className="arch-node__copy">
                      <strong>{node.title}</strong>
                      <small>{node.subtitle}</small>
                    </span>
                    {node.id === 'catalog' ? (
                      <span className="arch-node__side">
                        <Database aria-hidden="true" size={11} strokeWidth={1.8} />
                        Postgres
                      </span>
                    ) : null}
                  </button>
                  {index < FLOW.length - 1 ? (
                    <span aria-hidden="true" className="arch-arrow">
                      <span className="arch-arrow__line" />
                      <span className="arch-arrow__packet" />
                    </span>
                  ) : null}
                </div>
              );
            })}
          </div>

          <div className="arch-detail" aria-live="polite">
            <div className="arch-detail__head">
              <span className="arch-detail__label">selected piece</span>
              <span className="arch-detail__status">
                <span className="status-dot" />
                {selected.subtitle}
              </span>
            </div>
            <h3>{selected.title}</h3>
            <p>{selected.detail}</p>
          </div>
        </div>
      </Reveal>

      <Reveal className="arch-note" delay={0.1}>
        <p>
          The catalog never stores bundle bytes. It stores the release record, the manifest digest,
          and the install policy. The npm registry stores the tarball.
        </p>
      </Reveal>
    </section>
  );
}
