import { ArrowRight, ArrowUpRight } from 'lucide-react';
import { REPO_URL } from '../lib/content';
import { useI18n } from '../lib/i18n';
import { Magnet } from './ui/Magnet';
import { Reveal } from './ui/BlurText';

export function FinalCta() {
  const { content } = useI18n();
  const copy = content.finalCta;

  return (
    <section aria-labelledby="final-cta-title" className="final-cta">
      <Reveal>
        <div className="final-cta__inner">
          <p className="eyebrow">{copy.eyebrow}</p>
          <h2 id="final-cta-title">{copy.title}</h2>
          <p>{copy.body}</p>
          <div className="final-cta__actions">
            <Magnet>
              <a className="button button--primary" href="#quick-start">
                {copy.quickStart}
                <ArrowRight aria-hidden="true" size={16} strokeWidth={2} />
              </a>
            </Magnet>
            <a className="button button--secondary" href={REPO_URL} rel="noreferrer" target="_blank">
              {copy.openGithub}
              <ArrowUpRight aria-hidden="true" size={16} strokeWidth={2} />
            </a>
          </div>
        </div>
      </Reveal>
    </section>
  );
}
