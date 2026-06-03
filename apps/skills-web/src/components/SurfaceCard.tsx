import type { PropsWithChildren, ReactNode } from 'react';
import clsx from '../vendor/clsx';

interface SurfaceCardProps extends PropsWithChildren {
  title?: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}

export function SurfaceCard({ title, description, action, className, children }: SurfaceCardProps) {
  return (
    <section className={clsx('surface-card', className)}>
      {(title || description || action) && (
        <header className="surface-card__header">
          <div>
            {title ? <h2>{title}</h2> : null}
            {description ? <p>{description}</p> : null}
          </div>
          {action ? <div>{action}</div> : null}
        </header>
      )}
      {children}
    </section>
  );
}
