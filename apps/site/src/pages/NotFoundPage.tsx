import { Link } from 'react-router-dom';
import { BrandMark } from '../components/BrandMark';

export function NotFoundPage() {
  return (
    <div className="not-found">
      <span className="not-found__mark">
        <BrandMark />
      </span>
      <p className="eyebrow">404</p>
      <h1>Page not found</h1>
      <p className="lede">That route is not part of the site. Head back to the product page.</p>
      <Link className="button button--primary" to="/">
        Return home
      </Link>
    </div>
  );
}
