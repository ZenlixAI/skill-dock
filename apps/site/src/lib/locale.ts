export const LOCALES = ['en', 'zh'] as const;

export type Locale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: Locale = 'en';

export function isLocale(value: string | null | undefined): value is Locale {
  return value === 'en' || value === 'zh';
}

export function localeFromPathname(pathname: string): Locale {
  return pathname === '/zh' || pathname.startsWith('/zh/') ? 'zh' : DEFAULT_LOCALE;
}

export function stripLocalePrefix(pathname: string): string {
  let bare = pathname;
  if (pathname === '/zh') {
    bare = '/';
  } else if (pathname.startsWith('/zh/')) {
    bare = pathname.slice('/zh'.length);
  }
  return bare.length > 1 ? bare.replace(/\/+$/, '') : bare;
}

function splitSuffix(path: string): [string, string] {
  const index = path.search(/[?#]/);
  if (index === -1) return [path, ''];
  return [path.slice(0, index), path.slice(index)];
}

export function localizePath(path: string, locale: Locale): string {
  if (/^(?:[a-z][a-z\d+.-]*:)?\/\//i.test(path) || path.startsWith('#')) {
    return path;
  }

  const [pathname, suffix] = splitSuffix(path);
  const bare = stripLocalePrefix(pathname || '/');

  if (locale === 'en') {
    return `${bare || '/'}${suffix}`;
  }

  if (bare === '/') {
    return `/zh${suffix}`;
  }

  return `/zh${bare.startsWith('/') ? bare : `/${bare}`}${suffix}`;
}
