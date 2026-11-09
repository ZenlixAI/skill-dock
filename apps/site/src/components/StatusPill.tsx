import type { ReactNode } from 'react';
import clsx from '../vendor/clsx';

type Tone = 'green' | 'amber' | 'red' | 'blue' | 'muted';

interface StatusPillProps {
  tone?: Tone;
  children: ReactNode;
  className?: string;
}

export function StatusPill({ tone = 'green', children, className }: StatusPillProps) {
  return <span className={clsx('status-pill', `status-pill--${tone}`, className)}>{children}</span>;
}
