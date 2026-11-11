import { Check, Monitor, Moon, Sun } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useI18n } from '../lib/i18n';
import clsx from '../vendor/clsx';

type ThemePreference = 'system' | 'light' | 'dark';
type ResolvedTheme = 'light' | 'dark';

const STORAGE_KEY = 'skill-dock-theme';

function getSystemTheme(): ResolvedTheme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function resolveTheme(preference: ThemePreference): ResolvedTheme {
  return preference === 'system' ? getSystemTheme() : preference;
}

function applyTheme(preference: ThemePreference): ResolvedTheme {
  const resolved = resolveTheme(preference);
  document.documentElement.dataset.theme = resolved;
  document.documentElement.style.colorScheme = resolved;
  document.querySelectorAll('meta[name="theme-color"]').forEach((meta) => {
    meta.setAttribute('content', resolved === 'dark' ? '#070908' : '#f6f8f4');
  });
  return resolved;
}

function readPreference(): ThemePreference {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'light' || stored === 'dark' || stored === 'system') {
      return stored;
    }
  } catch {
    // Storage can be unavailable in private browsing contexts.
  }
  return 'system';
}

export function ThemeToggle() {
  const { content } = useI18n();
  const copy = content.ui.theme;
  const [preference, setPreference] = useState<ThemePreference>(readPreference);
  const [resolved, setResolved] = useState<ResolvedTheme>(() => resolveTheme(readPreference()));
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const options: Array<{
    id: ThemePreference;
    label: string;
    icon: typeof Monitor;
  }> = [
    { id: 'system', label: copy.system, icon: Monitor },
    { id: 'light', label: copy.light, icon: Sun },
    { id: 'dark', label: copy.dark, icon: Moon },
  ];

  useEffect(() => {
    setResolved(applyTheme(preference));

    if (preference !== 'system') return;

    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => setResolved(applyTheme('system'));
    media.addEventListener('change', handleChange);
    return () => media.removeEventListener('change', handleChange);
  }, [preference]);

  useEffect(() => {
    if (!open) return;

    const handlePointer = (event: PointerEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    };

    document.addEventListener('pointerdown', handlePointer);
    document.addEventListener('keydown', handleKey);
    requestAnimationFrame(() => {
      containerRef.current
        ?.querySelector<HTMLButtonElement>('[aria-checked="true"]')
        ?.focus();
    });

    return () => {
      document.removeEventListener('pointerdown', handlePointer);
      document.removeEventListener('keydown', handleKey);
    };
  }, [open]);

  function choose(next: ThemePreference) {
    setPreference(next);
    try {
      if (next === 'system') {
        localStorage.removeItem(STORAGE_KEY);
      } else {
        localStorage.setItem(STORAGE_KEY, next);
      }
    } catch {
      // The theme still applies for the current page.
    }
    setOpen(false);
  }

  const CurrentIcon = preference === 'system' ? Monitor : resolved === 'dark' ? Moon : Sun;
  const currentLabel =
    preference === 'system'
      ? copy.systemWith(resolved)
      : preference === 'dark'
        ? copy.dark
        : copy.light;

  return (
    <div className="theme-toggle" ref={containerRef}>
      <button
        aria-expanded={open}
        aria-haspopup="menu"
        aria-label={copy.trigger(currentLabel)}
        className={clsx('icon-button', open && 'icon-button--active')}
        onClick={() => setOpen((value) => !value)}
        type="button"
      >
        <CurrentIcon aria-hidden="true" size={16} strokeWidth={1.8} />
      </button>

      {open ? (
        <div aria-label={copy.label} className="theme-menu" role="menu">
          {options.map(({ id, label, icon: Icon }) => {
            const checked = preference === id;
            return (
              <button
                aria-checked={checked}
                className={clsx('theme-menu__item', checked && 'theme-menu__item--active')}
                key={id}
                onClick={() => choose(id)}
                role="menuitemradio"
                type="button"
              >
                <Icon aria-hidden="true" size={15} strokeWidth={1.8} />
                <span>{label}</span>
                {checked ? <Check aria-hidden="true" className="theme-menu__check" size={14} /> : null}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
