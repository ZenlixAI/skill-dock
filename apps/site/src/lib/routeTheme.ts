export type RouteTone = 'frontstage' | 'backstage';
export type RouteAccent = 'landing' | 'detail' | 'legal' | 'neutral';

export interface RouteTheme {
  tone: RouteTone;
  accent: RouteAccent;
}

export function getRouteTheme(pathname: string): RouteTheme {
  if (pathname === '/') {
    return {
      tone: 'frontstage',
      accent: 'landing',
    };
  }

  if (pathname === '/features' || pathname === '/docs') {
    return {
      tone: 'frontstage',
      accent: 'detail',
    };
  }

  if (pathname === '/privacy' || pathname === '/terms') {
    return {
      tone: 'frontstage',
      accent: 'legal',
    };
  }

  return {
    tone: 'frontstage',
    accent: 'neutral',
  };
}
