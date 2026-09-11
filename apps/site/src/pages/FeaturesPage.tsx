import { Search, Package, ShieldCheck } from 'lucide-react';
import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

const FEATURES = [
  {
    icon: Search,
    title: 'Find skills',
    description: 'Search the catalog from the CLI and get the exact package and version your team published.',
  },
  {
    icon: Package,
    title: 'Install from npm or Git',
    description: 'Install a released package, or select a skill directly from a local folder or Git repository.',
  },
  {
    icon: ShieldCheck,
    title: 'Keep releases controlled',
    description: 'Catalog owns package identity, versions, access, and release records while Artifact Keeper stores bundles.',
  },
] as const;

const COMMANDS = [
  'npx skills@latest find <query>',
  'npx skills@latest add <package-name>',
  'npx skills@latest add <git-url> --skill <name>',
  'npx skills@latest update',
  'npx skills@latest sync',
].join('\n');

export function FeaturesPage() {
  return (
    <div className="page-grid page-grid--detail features-page">
      <section className="hero-shell hero-shell--split features-page__hero">
        <div className="hero-shell__copy features-page__copy">
          <SectionHeading
            title="How it works"
            description="A private package registry for reusable agent skills."
            tone="frontstage"
          />
          <p className="features-page__lede">
            The server keeps the catalog and release decisions in one place. The CLI keeps installation simple for developers.
          </p>
        </div>

        <SurfaceCard
          className="features-page__storage-card"
          description="The public npm path and Catalog API are exposed by the same deployment."
          title="One endpoint, two jobs"
          variant="hero"
        >
          <dl className="features-page__storage-list">
            <div>
              <dt>Catalog</dt>
              <dd>Search, authentication, package identity, and release resolution.</dd>
            </div>
            <div>
              <dt>Artifact storage</dt>
              <dd>Private npm-compatible storage for package metadata and tarballs.</dd>
            </div>
            <div>
              <dt>CLI</dt>
              <dd>Find, install, publish, import, update, and sync from the terminal.</dd>
            </div>
          </dl>
        </SurfaceCard>
      </section>

      <section className="features-page__section">
        <div className="features-page__section-head">
          <h2>What developers can do</h2>
          <p>Most work happens through the CLI. You do not need to manage the artifact repository directly.</p>
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

      <section className="features-page__section features-page__section--sync">
        <div className="features-page__section-head">
          <h2>Automatic source publication</h2>
          <p>
            When <code>syncOnAdd</code> is enabled, adding a local or Git source installs it first and then sends a copy through Catalog for publication. A failed publication does not remove the local install.
          </p>
        </div>

        <SurfaceCard
          className="features-page__sync-config"
          description="Configure this globally, or override it per project."
          title="Global configuration"
          variant="subtle"
        >
          <pre className="features-page__code-block">
            <code>{'{\n  "syncOnAdd": true,\n  "packageNamePrefix": "@acme.skill/",\n  "defaultVersion": "0.1.0"\n}'}</code>
          </pre>
          <p className="features-page__sync-note">
            Stored at <code>${'{'}XDG_CONFIG_HOME:-~/.config{'}'}/skills/config.json</code>. A nearest project <code>skills.config.json</code> overrides global values.
          </p>
        </SurfaceCard>
      </section>

      <section className="features-page__section">
        <div className="features-page__section-head">
          <h2>Command model</h2>
          <p>Use released packages for repeatable installs; use source commands when you are developing or importing a skill.</p>
        </div>
        <SurfaceCard className="features-page__sync-config" description="See the repository README for installation and the full command reference." title="Common commands" variant="subtle">
          <pre className="features-page__code-block">
            <code>{COMMANDS}</code>
          </pre>
        </SurfaceCard>
      </section>
    </div>
  );
}
