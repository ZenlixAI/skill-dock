import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function PrivacyPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Legal"
        title="Privacy"
        description="This internal registry records the operational data needed to secure the catalog and audit change history."
        tone="legal"
      />

      <SurfaceCard
        title="What we collect"
        description="Registry activity is logged for security, package provenance, and operational support."
        variant="subtle"
      >
        <div className="stack-column">
          <p>We store account identity, session posture, package actions, and API activity required to operate the registry.</p>
          <p>Audit data is retained to investigate publishing events, governance changes, and access-control decisions.</p>
          <p>Do not store personal, customer, or regulated data inside package metadata unless the platform policy explicitly permits it.</p>
        </div>
      </SurfaceCard>
    </div>
  );
}
