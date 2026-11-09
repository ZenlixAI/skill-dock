import { Link } from 'react-router-dom';
export function NotFoundPage() {
  return (
    <div className="not-found">
      <p className="eyebrow">404</p>
      <h1>Page not found</h1>
      <p className="lede">That route is not part of the site. Head back to the product page.</p>
      <Link className="button button--primary" to="/">
        Return home
      </Link>
    </div>
  );
}
