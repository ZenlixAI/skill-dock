import { Check, Languages } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useI18n } from '../lib/i18n';
import { localizePath } from '../lib/locale';
import clsx from '../vendor/clsx';

export function LanguageToggle() {
  const { locale, content } = useI18n();
  const location = useLocation();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const current = content.language.options.find((option) => option.id === locale);
  const currentLabel = current?.label ?? content.language.options[0].label;
  const currentPath = `${location.pathname}${location.search}${location.hash}`;

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
      containerRef.current?.querySelector<HTMLAnchorElement>('[aria-current="true"]')?.focus();
    });

    return () => {
      document.removeEventListener('pointerdown', handlePointer);
      document.removeEventListener('keydown', handleKey);
    };
  }, [open]);

  return (
    <div className="language-toggle" ref={containerRef}>
      <button
        aria-expanded={open}
        aria-haspopup="menu"
        aria-label={content.language.switchLabel(currentLabel)}
        className={clsx('icon-button language-toggle__button', open && 'icon-button--active')}
        onClick={() => setOpen((value) => !value)}
        type="button"
      >
        <Languages aria-hidden="true" size={16} strokeWidth={1.8} />
        <span className="language-toggle__label">{locale === 'zh' ? '中文' : 'EN'}</span>
      </button>

      {open ? (
        <div aria-label={content.language.label} className="language-menu" role="menu">
          {content.language.options.map((option) => {
            const checked = option.id === locale;
            return (
              <Link
                aria-current={checked ? 'true' : undefined}
                className={clsx('language-menu__item', checked && 'language-menu__item--active')}
                key={option.id}
                lang={option.id === 'zh' ? 'zh-CN' : 'en'}
                onClick={() => setOpen(false)}
                role="menuitem"
                to={localizePath(currentPath, option.id)}
              >
                <span>{option.label}</span>
                {checked ? <Check aria-hidden="true" className="language-menu__check" size={14} /> : null}
              </Link>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
