export type RouteTone = 'frontstage' | 'backstage';
export type RouteAccent = 'landing' | 'detail' | 'console' | 'legal' | 'neutral';

export interface RouteTheme {
  tone: RouteTone;
  accent: RouteAccent;
}

const skillDetailPattern = /^\/skills\/[^/]+$/;
const versionDetailPattern = /^\/skills\/[^/]+\/versions\/[^/]+$/;

export function getRouteTheme(pathname: string): RouteTheme {
  if (pathname === '/' || pathname === '/catalog') {
    return {
      tone: 'frontstage',
      accent: pathname === '/' ? 'landing' : 'detail',
    };
  }

  if (skillDetailPattern.test(pathname) || versionDetailPattern.test(pathname)) {
    return {
      tone: 'frontstage',
      accent: 'detail',
    };
  }

  if (pathname === '/workbench') {
    return {
      tone: 'backstage',
      accent: 'detail',
    };
  }

  if (pathname === '/privacy' || pathname === '/terms') {
    return {
      tone: 'frontstage',
      accent: 'legal',
    };
  }

  if (pathname === '/login') {
    return {
      tone: 'backstage',
      accent: 'console',
    };
  }

  if (pathname !== '/workbench/import' && pathname !== '/workbench/authoring' && pathname !== '/workbench/governance' && pathname !== '/workbench/audit') {
    return {
      tone: 'backstage',
      accent: 'neutral',
    };
  }

  return {
    tone: 'backstage',
    accent: 'console',
  };
}
