import { createContext, useContext } from 'react';
import type { PropsWithChildren } from 'react';
import { useLocation } from 'react-router-dom';
import { getContent } from './content';
import type { SiteContent } from './content';
import { localeFromPathname, localizePath } from './locale';
import type { Locale } from './locale';

interface LocaleContextValue {
  locale: Locale;
}

const LocaleContext = createContext<LocaleContextValue>({ locale: 'en' });

export function LocaleProvider({ children }: PropsWithChildren) {
  const location = useLocation();
  const locale = localeFromPathname(location.pathname);

  return <LocaleContext.Provider value={{ locale }}>{children}</LocaleContext.Provider>;
}

export function useLocale(): Locale {
  return useContext(LocaleContext).locale;
}

export interface I18nValue {
  locale: Locale;
  content: SiteContent;
  path: (path: string) => string;
}

export function useI18n(): I18nValue {
  const locale = useLocale();

  return {
    locale,
    content: getContent(locale),
    path: (path: string) => localizePath(path, locale),
  };
}
