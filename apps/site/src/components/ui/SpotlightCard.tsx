import type { PropsWithChildren } from 'react';
import clsx from '../../vendor/clsx';

interface SpotlightCardProps extends PropsWithChildren {
  className?: string;
}

export function SpotlightCard({ children, className }: SpotlightCardProps) {
  function handlePointerMove(event: React.PointerEvent<HTMLDivElement>) {
    const rect = event.currentTarget.getBoundingClientRect();
    event.currentTarget.style.setProperty('--spot-x', `${event.clientX - rect.left}px`);
    event.currentTarget.style.setProperty('--spot-y', `${event.clientY - rect.top}px`);
  }

  return (
    <div className={clsx('spotlight-card', className)} onPointerMove={handlePointerMove}>
      {children}
    </div>
  );
}
