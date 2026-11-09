import { WORKFLOWS } from '../lib/content';
import { SectionHeader } from './SectionHeader';
import { Reveal } from './ui/BlurText';
import { SegmentedTabs } from './ui/SegmentedTabs';
import { Terminal } from './ui/Terminal';

export function HowItWorks() {
  const items = WORKFLOWS.map((workflow) => ({
    id: workflow.id,
    label: workflow.label,
    content: (
      <div className="workflow">
        <div className="workflow__copy">
          <h3>{workflow.title}</h3>
          <p>{workflow.body}</p>
          <div aria-label={`${workflow.label} flow`} className="workflow__flow" role="group">
            {workflow.flow.map((step, index) => (
              <span className="workflow__step" key={step}>
                {step}
                {index < workflow.flow.length - 1 ? (
                  <span aria-hidden="true" className="workflow__arrow">
                    →
                  </span>
                ) : null}
              </span>
            ))}
          </div>
        </div>
        <Terminal
          command={workflow.command}
          secondary={workflow.secondary ? `Or: ${workflow.secondary}` : undefined}
          title={workflow.label.toLowerCase()}
        />
      </div>
    ),
  }));

  return (
    <section aria-labelledby="how-it-works-title" className="section how-it-works" id="how-it-works">
      <Reveal>
        <SectionHeader
          eyebrow="How it works"
          id="how-it-works-title"
          lede="The catalog sits between the CLI and your npm registry. It decides what a version means and who can install it."
          title="Publish, resolve, govern."
        />
      </Reveal>
      <Reveal delay={0.05}>
        <SegmentedTabs ariaLabel="Release workflow" items={items} />
      </Reveal>
    </section>
  );
}
