import { Check } from 'lucide-react';
import { SectionHeader } from './SectionHeader';
import { Reveal } from './ui/BlurText';
import { SegmentedTabs } from './ui/SegmentedTabs';
import { Terminal } from './ui/Terminal';

const CATALOG_STEPS = [
  {
    title: 'Clone and configure',
    body: 'Replace the sample passwords and tokens in .env before you start the stack.',
    command: 'git clone https://github.com/ZenlixAI/skill-dock.git\ncd skill-dock\ncp .env.example .env',
  },
  {
    title: 'Start the stack',
    body: 'Compose starts the catalog, Postgres, and a local npm registry.',
    command: 'docker compose up --build -d\ncurl -fsS http://localhost:8080/readyz',
  },
] as const;

const CLI_STEPS = [
  {
    title: 'Build the CLI',
    body: 'The CLI lives in the same repository. Install it globally after the build.',
    command:
      'npx -y pnpm@10.17.1 --dir apps/skills-cli install\nnpx -y pnpm@10.17.1 --dir apps/skills-cli build\nnpm install -g ./apps/skills-cli',
  },
  {
    title: 'Point it at the catalog',
    body: 'Use the publisher token for authoring. Use the reader token for installs.',
    command:
      'export SKILLS_REGISTRY_URL=http://localhost:8080\nexport SKILLS_REGISTRY_TOKEN=<publisher-token>',
  },
  {
    title: 'Publish and install',
    body: 'The example uses the scoped package name the catalog expects.',
    command:
      'skills import-local ./my-skill \\\n  --package-name @acme.skill/my-skill \\\n  --version 1.4.0\n\nskills add @acme.skill/my-skill',
  },
] as const;

function Steps({ steps }: { steps: readonly { title: string; body: string; command: string }[] }) {
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
  return (
    <section aria-labelledby="quick-start-title" className="section quick-start" id="quick-start">
      <Reveal>
        <SectionHeader
          eyebrow="Quick start"
          id="quick-start-title"
          lede="The Compose stack starts the catalog, Postgres, and a local npm registry. You can swap the registry for Nexus or Artifactory when you deploy."
          title="Run it locally."
        />
      </Reveal>

      <Reveal delay={0.05}>
        <SegmentedTabs
          ariaLabel="Quick start path"
          items={[
            {
              id: 'catalog',
              label: 'Catalog',
              content: <Steps steps={CATALOG_STEPS} />,
            },
            {
              id: 'cli',
              label: 'CLI',
              content: <Steps steps={CLI_STEPS} />,
            },
          ]}
        />
      </Reveal>

      <Reveal className="quick-start__note" delay={0.1}>
        <p>
          <Check aria-hidden="true" size={15} strokeWidth={2.2} />
          For production, use <code>zarf/compose/production.yml</code>, set{' '}
          <code>SKILL_REGISTRY_ARTIFACT_READ_URL</code> to your npm registry, and put the catalog
          behind TLS.
        </p>
      </Reveal>
    </section>
  );
}
