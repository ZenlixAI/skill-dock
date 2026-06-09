import { Link } from 'react-router-dom';
import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function NotFoundPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="404"
        title="Page not found"
        description="The route you requested is not available in the registry. Use one of the portal entry points below."
        tone="legal"
      />

      <SurfaceCard
        action={
          <div className="button-row">
            <Link className="button button--primary" to="/">
              Back to overview
            </Link>
            <Link className="button button--secondary" to="/catalog">
              Browse catalog
            </Link>
          </div>
        }
        title="Return to the portal"
        description="Start from Overview for the product entry point, or jump into Catalog to continue discovery."
        variant="subtle"
      >
        <div className="stack-column">
          <div className="empty-box">Unknown routes no longer redirect silently. This page preserves context and gives a clear way back.</div>
        </div>
      </SurfaceCard>
    </div>
  );
}
