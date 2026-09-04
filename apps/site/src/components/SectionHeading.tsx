import type { ReactNode } from 'react';
import clsx from '../vendor/clsx';

interface SectionHeadingProps {
  title: string;
  description?: string;
  tone?: 'frontstage' | 'backstage' | 'detail' | 'legal';
  actions?: ReactNode;
}

export function SectionHeading({ title, description, tone = 'frontstage', actions }: SectionHeadingProps) {
  return (
    <section className={clsx('section-heading', `section-heading--${tone}`)}>
      <div className="section-heading__copy">
        <h1>{title}</h1>
        {description ? <p>{description}</p> : null}
      </div>
      {actions ? <div className="section-heading__actions">{actions}</div> : null}
    </section>
  );
}
