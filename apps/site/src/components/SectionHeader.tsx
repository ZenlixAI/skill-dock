import type { ReactNode } from 'react';
import clsx from '../vendor/clsx';

interface SectionHeaderProps {
  id?: string;
  eyebrow?: string;
  title: ReactNode;
  lede?: ReactNode;
  align?: 'left' | 'center';
}

export function SectionHeader({ id, eyebrow, title, lede, align = 'left' }: SectionHeaderProps) {
  return (
    <header className={clsx('section-header', `section-header--${align}`)}>
      {eyebrow ? <p className="eyebrow">{eyebrow}</p> : null}
      <h2 id={id}>{title}</h2>
      {lede ? <p className="lede">{lede}</p> : null}
    </header>
  );
}
