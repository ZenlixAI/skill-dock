import { Link } from 'react-router-dom';
import { LICENSE_URL } from '../lib/content';
import { useI18n } from '../lib/i18n';

export function TermsPage() {
  const { content, path } = useI18n();
  const copy = content.legal.terms;

  return (
    <div className="legal-page">
      <header className="legal-page__header">
        <p className="eyebrow">{content.legal.eyebrow}</p>
        <h1>{copy.title}</h1>
        <p className="lede">{copy.lede}</p>
      </header>

      <article aria-label={copy.articleLabel} className="legal-page__article">
        {copy.sections.map((section) => (
          <section key={section.heading}>
            <h2>{section.heading}</h2>
            <p>
              {section.body}
              {section.linkLabel ? <a href={LICENSE_URL}>{section.linkLabel}</a> : null}
              {section.bodyAfter ?? ''}
            </p>
          </section>
        ))}
      </article>

      <Link className="text-link" to={path('/')}>
        {content.legal.back}
      </Link>
    </div>
  );
}
