import { AnimatePresence, motion, useReducedMotion } from 'motion/react';
import { Plus } from 'lucide-react';
import { useState } from 'react';
import { FAQ } from '../lib/content';
import { SectionHeader } from './SectionHeader';
import { Reveal } from './ui/BlurText';
import clsx from '../vendor/clsx';

export function Faq() {
  const [open, setOpen] = useState<number | null>(0);
  const reduce = useReducedMotion();

  return (
    <section aria-labelledby="faq-title" className="section faq" id="faq">
      <Reveal>
        <SectionHeader
          eyebrow="FAQ"
          id="faq-title"
          lede="Short answers to the questions that come up before a first deployment."
          title="Questions that come up."
        />
      </Reveal>

      <Reveal delay={0.05}>
        <div className="faq-list">
          {FAQ.map((item, index) => {
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
                <AnimatePresence initial={false}>
                  {isOpen ? (
                    <motion.div
                      animate={{ height: 'auto', opacity: 1 }}
                      aria-labelledby={triggerId}
                      className="faq-panel"
                      exit={{ height: 0, opacity: 0 }}
                      id={panelId}
                      initial={reduce ? { height: 'auto', opacity: 1 } : { height: 0, opacity: 0 }}
                      role="region"
                      transition={{ duration: 0.28, ease: [0.16, 1, 0.3, 1] }}
                    >
                      <p>{item.answer}</p>
                    </motion.div>
                  ) : null}
                </AnimatePresence>
              </div>
            );
          })}
        </div>
      </Reveal>
    </section>
  );
}
