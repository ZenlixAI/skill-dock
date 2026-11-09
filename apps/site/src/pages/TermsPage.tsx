import { Link } from 'react-router-dom';

export function TermsPage() {
  return (
    <div className="legal-page">
      <header className="legal-page__header">
        <p className="eyebrow">Legal</p>
        <h1>Terms</h1>
        <p className="lede">Terms for using the Skill Dock website and related materials.</p>
      </header>

      <article aria-label="Terms content" className="legal-page__article">
        <section>
          <h2>Use of the site</h2>
          <p>
            The site provides product information, documentation links, and commands for running
            the software. You may use the site for evaluation and internal planning.
          </p>
        </section>

        <section>
          <h2>Software license</h2>
          <p>
            The Skill Dock source code is released under the MIT License. The license in the{' '}
            <a href="https://github.com/ZenlixAI/skill-dock/blob/master/LICENSE">repository</a>{' '}
            governs your use of the software.
          </p>
        </section>

        <section>
          <h2>Operational responsibility</h2>
          <p>
            You are responsible for the skills you publish, the registries you connect, and the
            access policies you apply. Review commands and configuration before you use them in a
            production environment.
          </p>
        </section>

        <section>
          <h2>No warranty</h2>
          <p>
            The site and the software are provided as is, without warranties of any kind. See the
            MIT License for the full disclaimer.
          </p>
        </section>
      </article>

      <Link className="text-link" to="/">
        Back to the product
      </Link>
    </div>
  );
}
