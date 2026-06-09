import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function TermsPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Legal"
        title="Terms"
        description="Use of the registry is limited to approved internal workflows for discovering, installing, and operating skills."
        tone="legal"
      />

      <SurfaceCard
        title="Usage expectations"
        description="These terms summarize the baseline rules for the portal and console."
        variant="subtle"
      >
        <div className="stack-column">
          <p>Only publish packages you are authorized to manage, and review install commands before distributing them to other teams.</p>
          <p>Governance and audit views are part of the product surface. Do not bypass them with undocumented operational flows.</p>
          <p>All usage remains subject to company security, compliance, and software supply-chain policies.</p>
        </div>
      </SurfaceCard>
    </div>
  );
}
