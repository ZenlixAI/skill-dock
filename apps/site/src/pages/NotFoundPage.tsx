import { Link } from 'react-router-dom';
import { useI18n } from '../lib/i18n';

export function NotFoundPage() {
  const { content, path } = useI18n();
  const copy = content.notFound;

  return (
    <div className="not-found">
      <p className="eyebrow">{copy.eyebrow}</p>
      <h1>{copy.title}</h1>
      <p className="lede">{copy.lede}</p>
      <Link className="button button--primary" to={path('/')}>
        {copy.back}
      </Link>
    </div>
  );
}
