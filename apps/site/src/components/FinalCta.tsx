import { ArrowRight, ArrowUpRight } from 'lucide-react';
import { REPO_URL } from '../lib/content';
import { Magnet } from './ui/Magnet';
import { Reveal } from './ui/BlurText';

export function FinalCta() {
  return (
    <section aria-labelledby="final-cta-title" className="final-cta">
      <Reveal>
        <div className="final-cta__inner">
          <p className="eyebrow">Start here</p>
          <h2 id="final-cta-title">Start with the catalog.</h2>
          <p>
            The repository has a Compose stack, a production Compose file, and the CLI source. No
            hosted account is required.
          </p>
          <div className="final-cta__actions">
            <Magnet>
              <a className="button button--primary" href="#quick-start">
                Quick start
                <ArrowRight aria-hidden="true" size={16} strokeWidth={2} />
              </a>
            </Magnet>
            <a className="button button--secondary" href={REPO_URL} rel="noreferrer" target="_blank">
              Open GitHub
              <ArrowUpRight aria-hidden="true" size={16} strokeWidth={2} />
            </a>
          </div>
        </div>
      </Reveal>
    </section>
  );
}
