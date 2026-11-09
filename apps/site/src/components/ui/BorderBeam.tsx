import clsx from '../../vendor/clsx';

interface BorderBeamProps {
  className?: string;
  duration?: number;
  delay?: number;
  size?: number;
}

export function BorderBeam({
  className,
  duration = 9,
  delay = 0,
  size = 140,
}: BorderBeamProps) {
  return (
    <span
      aria-hidden="true"
      className={clsx('border-beam', className)}
      style={
        {
          '--beam-duration': `${duration}s`,
          '--beam-delay': `${delay}s`,
          '--beam-size': `${size}px`,
        } as React.CSSProperties
      }
    />
  );
}
