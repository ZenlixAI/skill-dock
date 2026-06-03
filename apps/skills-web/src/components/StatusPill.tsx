import type { ReactNode } from 'react';
import clsx from '../vendor/clsx';

interface StatusPillProps {
  tone?: 'default' | 'good' | 'warn' | 'bad' | 'accent';
  children: ReactNode;
}

export function StatusPill({ tone = 'default', children }: StatusPillProps) {
  return <span className={clsx('status-pill', `status-pill--${tone}`)}>{children}</span>;
}
