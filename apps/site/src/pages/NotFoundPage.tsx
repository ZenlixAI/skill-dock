import { Link } from 'react-router-dom';
import { SectionHeading } from '../components/SectionHeading';
import { SurfaceCard } from '../components/SurfaceCard';

export function NotFoundPage() {
  return (
    <div className="page-grid">
      <SectionHeading
        title="Page not found"
        description="That route is no longer available. Return home or read the product features."
        tone="legal"
      />

      <SurfaceCard
        action={
          <div className="button-row">
            <Link className="button button--primary" to="/">
              Return home
            </Link>
            <Link className="button button--secondary" to="/features">
              Read features
            </Link>
          </div>
        }
        title="Back to the site"
        description="Public site routes keep the product overview, docs, and legal pages in one place."
        variant="subtle"
      >
        <div className="stack-column">
          <div className="empty-box">This page does not exist in the public site map. Use the links above to continue.</div>
        </div>
      </SurfaceCard>
    </div>
  );
}
