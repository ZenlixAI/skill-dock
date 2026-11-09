import { useId } from 'react';
import clsx from '../../vendor/clsx';

const SQUARES = [
  [3, 2],
  [7, 4],
  [11, 1],
  [15, 5],
  [5, 8],
  [12, 9],
  [18, 3],
  [9, 6],
] as const;

interface AnimatedGridPatternProps {
  className?: string;
}

export function AnimatedGridPattern({ className }: AnimatedGridPatternProps) {
  const rawId = useId();
  const patternId = `grid-${rawId.replace(/[^a-zA-Z0-9]/g, '')}`;

  return (
    <div aria-hidden="true" className={clsx('grid-pattern', className)}>
      <svg preserveAspectRatio="none" viewBox="0 0 1200 620">
        <defs>
          <pattern height="44" id={patternId} patternUnits="userSpaceOnUse" width="44">
            <path d="M44 0H0V44" fill="none" stroke="currentColor" strokeWidth="1" />
          </pattern>
        </defs>
        <rect fill={`url(#${patternId})`} height="620" width="1200" />
        {SQUARES.map(([x, y], index) => (
          <rect
            className="grid-pattern__square"
            fill="currentColor"
            height="44"
            key={`${x}-${y}`}
            style={{ animationDelay: `${index * 0.9}s` }}
            width="44"
            x={x * 44}
            y={y * 44}
          />
        ))}
      </svg>
      <span className="grid-pattern__glow" />
    </div>
  );
}
