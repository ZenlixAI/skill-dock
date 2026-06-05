import type { ReactNode } from 'react';
import clsx from '../vendor/clsx';

interface SectionHeadingProps {
  eyebrow?: string;
  title: string;
  description?: string;
  tone?: 'frontstage' | 'backstage' | 'detail';
  actions?: ReactNode;
}

export function SectionHeading({ eyebrow, title, description, tone = 'frontstage', actions }: SectionHeadingProps) {
  return (
    <section className={clsx('section-heading', `section-heading--${tone}`)}>
      <div>
        {eyebrow ? <span className="eyebrow">{eyebrow}</span> : null}
        <h1>{title}</h1>
        {description ? <p>{description}</p> : null}
      </div>
      {actions ? <div className="section-heading__actions">{actions}</div> : null}
    </section>
  );
}
