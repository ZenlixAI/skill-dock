import { useI18n } from '../lib/i18n';
import { SectionHeader } from './SectionHeader';
import { Reveal } from './ui/BlurText';
import { SegmentedTabs } from './ui/SegmentedTabs';
import { Terminal } from './ui/Terminal';

export function HowItWorks() {
  const { content } = useI18n();
  const copy = content.howItWorks;

  const items = copy.workflows.map((workflow) => ({
    id: workflow.id,
    label: workflow.label,
    content: (
      <div className="workflow">
        <div className="workflow__copy">
          <h3>{workflow.title}</h3>
          <p>{workflow.body}</p>
          <div aria-label={copy.flowLabel(workflow.label)} className="workflow__flow" role="group">
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
          secondary={workflow.secondary ? `${copy.orPrefix}${workflow.secondary}` : undefined}
          title={workflow.label.toLowerCase()}
        />
      </div>
    ),
  }));

  return (
    <section aria-labelledby="how-it-works-title" className="section how-it-works" id="how-it-works">
      <Reveal>
        <SectionHeader
          eyebrow={copy.eyebrow}
          id="how-it-works-title"
          lede={copy.lede}
          title={copy.title}
        />
      </Reveal>
      <Reveal delay={0.05}>
        <SegmentedTabs ariaLabel={copy.tablistLabel} items={items} />
      </Reveal>
    </section>
  );
}
