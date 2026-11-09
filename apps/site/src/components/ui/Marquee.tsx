import clsx from '../../vendor/clsx';

interface MarqueeProps {
  items: readonly string[];
  className?: string;
  duration?: number;
  reverse?: boolean;
}

export function Marquee({ items, className, duration = 34, reverse = false }: MarqueeProps) {
  return (
    <div className={clsx('marquee', reverse && 'marquee--reverse', className)}>
      <div
        className="marquee__track"
        style={{ '--marquee-duration': `${duration}s` } as React.CSSProperties}
      >
        <ul className="marquee__group">
          {items.map((item) => (
            <li className="marquee__item" key={item}>
              <span className="marquee__dot" />
              {item}
            </li>
          ))}
        </ul>
        <ul aria-hidden="true" className="marquee__group">
          {items.map((item) => (
            <li className="marquee__item" key={`${item}-duplicate`}>
              <span className="marquee__dot" />
              {item}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
