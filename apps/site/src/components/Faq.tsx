import { motion, useReducedMotion } from 'motion/react';
import { Plus } from 'lucide-react';
import { useState } from 'react';
import { useI18n } from '../lib/i18n';
import { SectionHeader } from './SectionHeader';
import { Reveal } from './ui/BlurText';
import clsx from '../vendor/clsx';

export function Faq() {
  const { content } = useI18n();
  const copy = content.faq;
  const [open, setOpen] = useState<number | null>(0);
  const reduce = useReducedMotion();

  return (
    <section aria-labelledby="faq-title" className="section faq" id="faq">
      <Reveal>
        <SectionHeader
          eyebrow={copy.eyebrow}
          id="faq-title"
          lede={copy.lede}
          title={copy.title}
        />
      </Reveal>

      <Reveal delay={0.05}>
        <div className="faq-list">
          {copy.items.map((item, index) => {
            const isOpen = open === index;
            const panelId = `faq-panel-${index}`;
            const triggerId = `faq-trigger-${index}`;
            return (
              <div className="faq-item" key={item.question}>
                <h3>
                  <button
                    aria-controls={panelId}
                    aria-expanded={isOpen}
                    className={clsx('faq-trigger', isOpen && 'faq-trigger--open')}
                    id={triggerId}
                    onClick={() => setOpen(isOpen ? null : index)}
                    type="button"
                  >
                    <span>{item.question}</span>
                    <span className="faq-trigger__icon" aria-hidden="true">
                      <Plus size={16} strokeWidth={1.8} />
                    </span>
                  </button>
                </h3>
                <motion.div
                  animate={isOpen ? { height: 'auto', opacity: 1 } : { height: 0, opacity: 0 }}
                  aria-hidden={!isOpen}
                  aria-labelledby={triggerId}
                  className="faq-panel"
                  id={panelId}
                  initial={false}
                  role="region"
                  transition={reduce ? { duration: 0 } : { duration: 0.28, ease: [0.16, 1, 0.3, 1] }}
                >
                  <p>{item.answer}</p>
                </motion.div>
              </div>
            );
          })}
        </div>
      </Reveal>
    </section>
  );
}
