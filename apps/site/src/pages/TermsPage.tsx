import { SectionHeading } from '../components/SectionHeading';

export function TermsPage() {
  return (
    <div className="page-shell page-shell--enter legal-page">
      <SectionHeading
        title="Terms"
        description="Skill Registry is for approved product evaluation, installation, and operations."
        tone="legal"
      />

      <article className="legal-page__article" aria-label="Terms content">
        <section className="legal-page__section">
          <h2>Usage requirements</h2>
          <div className="legal-page__prose">
            <p>Only publish materials you are authorized to manage.</p>
            <p>Review install commands before sharing them with other teams.</p>
            <p>Do not use undocumented workflows to bypass the public site or release flow.</p>
            <p>All usage must still comply with your company’s security, compliance, and software supply chain policies.</p>
          </div>
        </section>
      </article>
    </div>
  );
}
