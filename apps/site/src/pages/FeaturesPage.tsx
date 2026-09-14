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
    title: 'Share skills',
    description: 'Give developers and agents a consistent way to access the knowledge and workflows your team creates.',
  },
  {
    icon: ShieldCheck,
    title: 'Manage with confidence',
    description: 'Keep your skill library organized, dependable, and aligned with the standards your team cares about.',
  },
] as const;

const WORKFLOW = [
  {
    title: 'Find what you need',
    description: 'Browse a shared collection of skills designed around the work your team does every day.',
  },
  {
    title: 'Put it to work',
    description: 'Make useful capabilities available to the agents and people who need them.',
  },
  {
    title: 'Keep improving',
    description: 'Maintain a trusted library as your team learns, changes, and grows.',
  },
] as const;

export function FeaturesPage() {
  return (
    <div className="page-grid page-grid--detail features-page">
      <section className="features-page__intro">
        <SectionHeading
          title="How it works"
          description="A simple home for the skills your agents rely on."
          tone="frontstage"
        />
        <p className="features-page__lede">
          Skill Registry helps teams turn shared knowledge into reusable capabilities that are easy to discover, use, and maintain.
        </p>
      </section>

      <section className="features-page__section" aria-labelledby="features-title">
        <div className="features-page__section-head">
          <h2 id="features-title">Everything your team needs</h2>
          <p>Bring the most important parts of a shared skill library into one calm, predictable experience.</p>
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

      <section className="features-page__section" aria-labelledby="workflow-title">
        <div className="features-page__section-head">
          <h2 id="workflow-title">A shared foundation for agent work</h2>
          <p>Move from a one-off solution to a capability your whole team can use and improve.</p>
        </div>

        <div className="features-page__workflow">
          {WORKFLOW.map(({ title, description }, index) => (
            <article className="features-page__workflow-item" key={title}>
              <span aria-hidden="true">0{index + 1}</span>
              <div>
                <h3>{title}</h3>
                <p>{description}</p>
              </div>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
