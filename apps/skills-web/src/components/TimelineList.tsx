import type { ReactNode } from 'react';

interface TimelineItem {
  key: string;
  title: string;
  subtitle: string;
  meta?: ReactNode;
}

interface TimelineListProps {
  items: TimelineItem[];
}

export function TimelineList({ items }: TimelineListProps) {
  return (
    <div className="timeline-list">
      {items.map((item) => (
        <article className="timeline-item" key={item.key}>
          <div className="timeline-item__bar" />
          <div className="timeline-item__body">
            <strong>{item.title}</strong>
            <p>{item.subtitle}</p>
            {item.meta ? <div>{item.meta}</div> : null}
          </div>
        </article>
      ))}
    </div>
  );
}
