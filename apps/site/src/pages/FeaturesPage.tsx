import { Package, Search, ShieldCheck } from 'lucide-react';
import { SectionHeading } from '../components/SectionHeading';

const FEATURES = [
  {
    icon: Search,
    title: 'Discover skills',
    description: 'Help your team find the right capabilities quickly through a clear, focused library of reusable agent skills.',
  },
  {
    icon: Package,
    title: 'Use the familiar CLI',
    description: 'Replace the skills.sh CLI without changing how your team discovers, installs, and works with agent skills.',
  },
  {
    icon: ShieldCheck,
    title: 'Manage with confidence',
    description: 'Keep your skill library organized, dependable, and aligned with the standards your team cares about.',
  },
] as const;

const CLI_COMMANDS = [
  ['Find a skill', 'npx skills find <query>'],
  ['Install a skill', 'npx skills add <owner/repo>'],
  ['Update skills', 'npx skills update'],
  ['List installed skills', 'npx skills list'],
] as const;

export function FeaturesPage() {
  return (
    <div className="page-grid page-grid--detail features-page">
      <section className="features-page__intro">
        <SectionHeading
          title="How it works"
          description="A one-for-one replacement for the skills.sh CLI, with a private catalog your team controls."
          tone="frontstage"
        />
        <p className="features-page__lede">
          Skill Registry provides the same familiar CLI workflow as skills.sh, while giving teams a private place to discover, share, and manage skills.
        </p>
      </section>

      <section className="features-page__section" aria-labelledby="features-title">
        <div className="features-page__section-head">
          <h2 id="features-title">A one-for-one CLI replacement</h2>
          <p>Keep the workflow your team already knows while bringing skill discovery and management into your own environment.</p>
        </div>

        <div className="features-page__matrix">
          {FEATURES.map(({ icon: Icon, title, description }) => (
            <article className="features-page__column" key={title}>
              <Icon aria-hidden="true" size={18} strokeWidth={1.8} />
              <h3>{title}</h3>
              <p>{description}</p>
            </article>
          ))}
        </div>
      </section>

      <section className="features-page__section" aria-labelledby="cli-title">
        <div className="features-page__section-head">
          <h2 id="cli-title">Familiar commands, private control</h2>
          <p>Use the skills CLI workflow you already know. Point it at Skill Registry and keep the same everyday commands.</p>
        </div>

        <div className="features-page__cli-commands">
          {CLI_COMMANDS.map(([label, command]) => (
            <div className="features-page__cli-command" key={label}>
              <span>{label}</span>
              <code>{command}</code>
            </div>
          ))}
        </div>
      </section>


    </div>
  );
}
