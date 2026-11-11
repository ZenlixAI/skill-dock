import { AGENTS } from '../lib/content';
import { useI18n } from '../lib/i18n';
import { Marquee } from './ui/Marquee';

export function AgentStrip() {
  const { content } = useI18n();
  const copy = content.agents;

  return (
    <section aria-labelledby="agents-title" className="agent-strip">
      <div className="agent-strip__intro">
        <p className="eyebrow">{copy.eyebrow}</p>
        <h2 id="agents-title">{copy.title}</h2>
        <p className="agent-strip__copy">{copy.copy}</p>
      </div>
      <Marquee className="agent-strip__marquee" items={AGENTS} />
    </section>
  );
}
