export type RouteTone = 'frontstage' | 'backstage';
export type RouteAccent = 'landing' | 'detail' | 'console';

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

  return {
    tone: 'backstage',
    accent: 'console',
  };
}
