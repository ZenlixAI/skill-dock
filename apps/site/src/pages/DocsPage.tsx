import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';

const REQUIRED_SERVER_ENV = [
  'POSTGRES_DB=skill_registry',
  'POSTGRES_USER=skill_registry',
  'POSTGRES_PASSWORD=<generate-a-secure-password>',
  'CATALOG_PORT=8080',
  'SKILL_REGISTRY_READER_TOKEN=<generate-a-reader-token>',
  'SKILL_REGISTRY_AUTHORING_TOKEN=<generate-an-authoring-token>',
  'SKILL_REGISTRY_PUBLISHER_TOKEN=<generate-a-publisher-token>',
  'SKILL_REGISTRY_INSTALL_TARGET_MODE=proxy',
].join('\n');

const OPTIONAL_SERVER_ENV = [
  'SKILL_REGISTRY_ARTIFACT_READ_URL=http://artifact.example.internal:30080/',
  'SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION=Basic ...',
  'SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION=Basic ...',
  'SKILL_REGISTRY_PUBLIC_INSTALL_URL=http://127.0.0.1:8080/repository/skills-npm/',
  'SKILL_REGISTRY_AUTH_TOKENS_JSON={...}',
].join('\n');

const CLI_SYNC_CONFIG = [
  '{',
  '  "syncOnAdd": true,',
  '  "packageNamePrefix": "@acme.skill/",',
  '  "defaultVersion": "0.1.0",',
  '  "registry": "http://127.0.0.1:8080"',
  '}',
].join('\n');

const DEPLOY_STEPS = [
  {
    title: 'Start the production stack',
    description: 'Use the Docker Compose file to run PostgreSQL and the backend together.',
  },
  {
    title: 'Load server and storage config',
    description: 'Set the database URL, registry tokens, and artifact repository overrides in `.env` or the shell.',
  },
  {
    title: 'Point the CLI at the server',
    description: 'Set the registry URL and token before you search, install, publish, or import.',
  },
] as const;

export function DocsPage() {
  return (
    <div className="page-grid page-grid--detail docs-page">
      <nav className="docs-page__toc" aria-label="On this page">
        <strong>On this page</strong>
        <a href="#deploy">Deploy</a>
        <a href="#server">Server config</a>
        <a href="#cli">CLI config</a>
        <a href="#sync">Source sync</a>
      </nav>
      <section className="hero-shell hero-shell--split docs-page__hero">
        <div className="hero-shell__copy docs-page__copy">
          <SectionHeading
            title="Get started"
            description="Deploy the backend, connect the storage, and point the CLI at the same service."
            tone="detail"
          />

          <p className="docs-page__lede">
            Start with the deployment path, then copy the environment values that keep the backend, storage, CLI, and
            source-sync behavior aligned.
          </p>

          <div className="context-strip">
            <StatusPill tone="accent">Docker</StatusPill>
            <StatusPill>Configuration</StatusPill>
            <StatusPill>Backend</StatusPill>
          </div>
        </div>
      </section>

      <section className="docs-page__section" id="deploy">
        <div className="docs-page__section-head">
          <h2>Deploy the backend in three steps</h2>
          <p>Keep the production stack, the private storage, and the CLI pointed at one backend.</p>
        </div>

        <div className="docs-page__step-grid">
          {DEPLOY_STEPS.map((item, index) => (
            <article className="docs-page__step-card" key={item.title}>
              <p className="docs-page__step-kicker">0{index + 1}</p>
              <h3>{item.title}</h3>
              <p>{item.description}</p>
            </article>
          ))}
        </div>
      </section>

      <section className="docs-page__section docs-page__section--grid" id="server">
        <div className="docs-page__section-head">
          <h2>Configure the server</h2>
          <p>Set the backend through explicit environment variables instead of hidden image defaults.</p>
        </div>

        <div className="docs-page__config-grid">
          <SurfaceCard className="docs-page__panel" description="These values are required for the Docker deployment." title="Required configuration" variant="subtle">
            <pre className="docs-page__code-block">
              <code>{REQUIRED_SERVER_ENV}</code>
            </pre>
            <p className="docs-page__security-note"><strong>Before production:</strong> replace every placeholder with a unique secret. Keep reader, authoring, and publisher tokens scoped to their role and rotate them when access changes.</p>
          </SurfaceCard>

          <SurfaceCard className="docs-page__panel" description="Use these only when the artifact repository needs its own credentials or overrides." title="Repository and auth overrides" variant="subtle">
            <pre className="docs-page__code-block">
              <code>{OPTIONAL_SERVER_ENV}</code>
            </pre>
            <p className="docs-page__panel-note">
              Set these values in a `.env` file or export them before you run `docker compose`.
            </p>
          </SurfaceCard>
        </div>
      </section>

      <section className="docs-page__section" id="cli">
        <div className="docs-page__section-head">
          <h2>Configure the CLI</h2>
          <p>Point local installs, search, publish, import, and sync at the same backend.</p>
        </div>

        <SurfaceCard description="Use the same backend URL and token in the shell that runs the CLI." title="CLI runtime" variant="subtle">
          <div className="docs-page__cli-grid">
            <div className="docs-page__cli-block">
              <span>Registry URL</span>
              <code>export SKILLS_REGISTRY_URL=http://127.0.0.1:8080</code>
            </div>
            <div className="docs-page__cli-block">
              <span>Reader token</span>
              <code>export SKILLS_REGISTRY_TOKEN=&lt;your-reader-token&gt;</code>
            </div>
          </div>
          <p className="docs-page__panel-note">
            If the server returns `direct` install targets, also make the relevant npm credentials available through
            `NPM_CONFIG_USERCONFIG` or a local `.npmrc`.
          </p>
        </SurfaceCard>
      </section>

      <section className="docs-page__section" id="sync">
        <div className="docs-page__section-head">
          <h2>Sync source installs automatically</h2>
          <p>
            Source installs sync through the backend import flow by default. Add this project-level file when a package
            name prefix, default version, or registry needs to fill missing source metadata.
          </p>
        </div>

        <SurfaceCard
          className="docs-page__sync-card"
          description="Place this file in the directory where you run the CLI."
          title="skills.config.json"
          variant="subtle"
        >
          <pre className="docs-page__code-block">
            <code>{CLI_SYNC_CONFIG}</code>
          </pre>
          <div className="docs-page__sync-details">
            <p><code>syncOnAdd: true</code> makes the default explicit; set it to <code>false</code> to keep installs local only.</p>
            <p><code>packageNamePrefix</code> and <code>defaultVersion</code> fill missing source metadata.</p>
          </div>
        </SurfaceCard>
      </section>

      <p className="docs-page__closing">Deploy the backend with Docker, point the CLI at it, and keep storage private.</p>
    </div>
  );
}
