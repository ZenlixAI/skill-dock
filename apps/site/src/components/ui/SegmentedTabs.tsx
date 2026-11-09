import { motion } from 'motion/react';
import { useId, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import clsx from '../../vendor/clsx';

export interface TabItem {
  id: string;
  label: string;
  content: ReactNode;
}

interface SegmentedTabsProps {
  items: TabItem[];
  ariaLabel: string;
  className?: string;
}

export function SegmentedTabs({ items, ariaLabel, className }: SegmentedTabsProps) {
  const [active, setActive] = useState(items[0]?.id ?? '');
  const layoutId = useId().replace(/[^a-zA-Z0-9]/g, '');
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const activeItem = items.find((item) => item.id === active) ?? items[0];

  function moveFocus(index: number) {
    const next = items[index];
    if (!next) return;
    setActive(next.id);
    tabRefs.current[index]?.focus();
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    const currentIndex = items.findIndex((item) => item.id === active);
    if (event.key === 'ArrowRight') {
      event.preventDefault();
      moveFocus((currentIndex + 1) % items.length);
    }
    if (event.key === 'ArrowLeft') {
      event.preventDefault();
      moveFocus((currentIndex - 1 + items.length) % items.length);
    }
    if (event.key === 'Home') {
      event.preventDefault();
      moveFocus(0);
    }
    if (event.key === 'End') {
      event.preventDefault();
      moveFocus(items.length - 1);
    }
  }

  if (!activeItem) return null;

  return (
    <div className={clsx('tabs', className)}>
      <div
        aria-label={ariaLabel}
        className="tabs__list"
        onKeyDown={handleKeyDown}
        role="tablist"
      >
        {items.map((item, index) => {
          const selected = item.id === active;
          return (
            <button
              aria-controls={`panel-${layoutId}-${item.id}`}
              aria-selected={selected}
              className={clsx('tabs__button', selected && 'tabs__button--active')}
              id={`tab-${layoutId}-${item.id}`}
              key={item.id}
              onClick={() => setActive(item.id)}
              ref={(node) => {
                tabRefs.current[index] = node;
              }}
              role="tab"
              tabIndex={selected ? 0 : -1}
              type="button"
            >
              {selected ? (
                <motion.span
                  className="tabs__pill"
                  layoutId={`tab-pill-${layoutId}`}
                  transition={{ type: 'spring', stiffness: 380, damping: 32 }}
                />
              ) : null}
              <span className="tabs__label">{item.label}</span>
            </button>
          );
        })}
      </div>

      <div className="tabs__panels">
        {items.map((item) => {
          const selected = item.id === active;
          return (
            <div
              aria-labelledby={`tab-${layoutId}-${item.id}`}
              className={clsx('tabs__panel', selected && 'tabs__panel--active')}
              hidden={!selected}
              id={`panel-${layoutId}-${item.id}`}
              key={item.id}
              role="tabpanel"
              tabIndex={0}
            >
              {item.content}
            </div>
          );
        })}
      </div>
    </div>
  );
}
