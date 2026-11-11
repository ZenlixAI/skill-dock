import { Boxes, Database, Server, TerminalSquare } from 'lucide-react';
import { useState } from 'react';
import { useI18n } from '../lib/i18n';
import { SectionHeader } from './SectionHeader';
import { Reveal } from './ui/BlurText';
import { BorderBeam } from './ui/BorderBeam';

export function Architecture() {
  const { content } = useI18n();
  const copy = content.architecture;
  const [active, setActive] = useState('catalog');
  const selected = copy.nodes.find((node) => node.id === active) ?? copy.nodes[1];
  const flow = [
    { ...copy.nodes[0], icon: TerminalSquare },
    { ...copy.nodes[1], icon: Server },
    { ...copy.nodes[3], icon: Boxes },
  ];

  return (
    <section aria-labelledby="architecture-title" className="section architecture" id="architecture">
      <Reveal>
        <SectionHeader
          eyebrow={copy.eyebrow}
          id="architecture-title"
          lede={copy.lede}
          title={copy.title}
        />
      </Reveal>

      <Reveal delay={0.05}>
        <div className="arch-shell">
          <BorderBeam duration={12} size={180} />
          <div aria-label={copy.flowLabel} className="arch-flow" role="group">
            {flow.map((node, index) => {
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
                  {index < flow.length - 1 ? (
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
              <span className="arch-detail__label">{copy.selectedLabel}</span>
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
        <p>{copy.note}</p>
      </Reveal>
    </section>
  );
}
