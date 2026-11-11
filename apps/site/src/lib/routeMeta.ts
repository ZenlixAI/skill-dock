import { getContent } from './content';
import { localeFromPathname, stripLocalePrefix } from './locale';

export type RouteSection = 'portal' | 'legal' | 'not-found';

export interface RouteMeta {
  title: string;
  description: string;
  section: RouteSection;
}

export function getRouteMeta(pathname: string): RouteMeta {
  const content = getContent(localeFromPathname(pathname)).meta;
  const path = stripLocalePrefix(pathname);

  if (path === '/') {
    return { ...content.home, section: 'portal' };
  }

  if (path === '/registry') {
    return { ...content.registry, section: 'portal' };
  }

  if (path === '/privacy') {
    return { ...content.privacy, section: 'legal' };
  }

  if (path === '/terms') {
    return { ...content.terms, section: 'legal' };
  }

  return { ...content.notFound, section: 'not-found' };
}
