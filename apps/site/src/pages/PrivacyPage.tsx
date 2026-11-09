import { Link } from 'react-router-dom';

export function PrivacyPage() {
  return (
    <div className="legal-page">
      <header className="legal-page__header">
        <p className="eyebrow">Legal</p>
        <h1>Privacy</h1>
        <p className="lede">How the Skill Dock website handles site usage data.</p>
      </header>

      <article aria-label="Privacy content" className="legal-page__article">
        <section>
          <h2>What the site collects</h2>
          <p>
            The public site is a static build. It does not include an account system, analytics
            script, or advertising tracker. The hosting provider may keep standard request logs such
            as IP address, user agent, and requested path.
          </p>
        </section>

        <section>
          <h2>Clipboard actions</h2>
          <p>
            Copy buttons write the selected command to your clipboard through the browser API. The
            site does not send the copied text anywhere.
          </p>
        </section>

        <section>
          <h2>External links</h2>
          <p>
            Links to GitHub and the documentation open external sites. Their privacy policies apply
            once you leave this site.
          </p>
        </section>

        <section>
          <h2>Contact</h2>
          <p>
            For questions about the project, open an issue in the{' '}
            <a href="https://github.com/ZenlixAI/skill-dock/issues">GitHub repository</a>.
          </p>
        </section>
      </article>

      <Link className="text-link" to="/">
        Back to the product
      </Link>
    </div>
  );
}
