import type { ReactNode } from 'react';
import clsx from '../vendor/clsx';

interface StatusPillProps {
  tone?: 'default' | 'good' | 'warn' | 'bad' | 'accent';
  children: ReactNode;
  className?: string;
}

export function StatusPill({ tone = 'default', className, children }: StatusPillProps) {
  return <span className={clsx('status-pill', `status-pill--${tone}`, className)}>{children}</span>;
}
