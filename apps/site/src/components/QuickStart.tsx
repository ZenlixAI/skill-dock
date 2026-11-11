import { Check } from 'lucide-react';
import type { QuickStartStep } from '../lib/content';
import { useI18n } from '../lib/i18n';
import { SectionHeader } from './SectionHeader';
import { Reveal } from './ui/BlurText';
import { SegmentedTabs } from './ui/SegmentedTabs';
import { Terminal } from './ui/Terminal';

function Steps({ steps }: { steps: readonly QuickStartStep[] }) {
  return (
    <ol className="quick-steps">
      {steps.map((step, index) => (
        <li className="quick-step" key={step.title}>
          <div className="quick-step__number">
            <span>{String(index + 1).padStart(2, '0')}</span>
          </div>
          <div className="quick-step__copy">
            <h3>{step.title}</h3>
            <p>{step.body}</p>
          </div>
          <Terminal command={step.command} title={step.title.toLowerCase()} />
        </li>
      ))}
    </ol>
  );
}

export function QuickStart() {
  const { content } = useI18n();
  const copy = content.quickStart;

  return (
    <section aria-labelledby="quick-start-title" className="section quick-start" id="quick-start">
      <Reveal>
        <SectionHeader
          eyebrow={copy.eyebrow}
          id="quick-start-title"
          lede={copy.lede}
          title={copy.title}
        />
      </Reveal>

      <Reveal delay={0.05}>
        <SegmentedTabs
          ariaLabel={copy.tablistLabel}
          items={[
            {
              id: 'catalog',
              label: copy.catalogTab,
              content: <Steps steps={copy.catalogSteps} />,
            },
            {
              id: 'cli',
              label: copy.cliTab,
              content: <Steps steps={copy.cliSteps} />,
            },
          ]}
        />
      </Reveal>

      <Reveal className="quick-start__note" delay={0.1}>
        <p>
          <Check aria-hidden="true" size={15} strokeWidth={2.2} />
          {copy.productionNotePrefix}
          <code>zarf/compose/production.yml</code>
          {copy.productionNoteMiddle}
          <code>SKILL_REGISTRY_ARTIFACT_READ_URL</code>
          {copy.productionNoteSuffix}
        </p>
      </Reveal>
    </section>
  );
}
