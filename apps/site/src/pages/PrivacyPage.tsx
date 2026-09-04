import { SectionHeading } from '../components/SectionHeading';

export function PrivacyPage() {
  return (
    <div className="page-shell page-shell--enter legal-page">
      <SectionHeading
        title="Privacy"
        description="How Skill Registry handles site usage data and product interactions."
        tone="legal"
      />

      <article className="legal-page__article" aria-label="Privacy content">
        <section className="legal-page__section">
          <h2>What we collect</h2>
          <div className="legal-page__prose">
            <p>We may store site visits, copy actions, and basic request logs.</p>
            <p>If you open a page or copy a command, that interaction may be recorded as product usage.</p>
            <p>Please avoid placing personal, customer, or regulated data in site copy or examples.</p>
          </div>
        </section>
      </article>
    </div>
  );
}
