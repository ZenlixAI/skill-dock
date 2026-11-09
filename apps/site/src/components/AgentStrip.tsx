import { AGENTS } from '../lib/content';
import { Marquee } from './ui/Marquee';

export function AgentStrip() {
  return (
    <section aria-labelledby="agents-title" className="agent-strip">
      <div className="agent-strip__intro">
        <p className="eyebrow">Agent coverage</p>
        <h2 id="agents-title">
          One CLI, 56 agent targets.
        </h2>
        <p className="agent-strip__copy">
          The same bundle installs into Claude Code, Codex, Cursor, OpenCode, and the directories
          other agents read. Choose project or global scope.
        </p>
      </div>
      <Marquee className="agent-strip__marquee" items={AGENTS} />
    </section>
  );
}
