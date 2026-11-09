export type RouteSection = 'portal' | 'legal' | 'not-found';

export interface RouteMeta {
  title: string;
  description: string;
  section: RouteSection;
}

export function getRouteMeta(pathname: string): RouteMeta {
  if (pathname === '/') {
    return {
      title: 'Skill Registry · A private catalog for agent skills',
      description:
        'A self-hosted catalog for agent skills. Store bundles in your npm registry, publish exact versions, resolve installs, and govern releases.',
      section: 'portal',
    };
  }

  if (pathname === '/privacy') {
    return {
      title: 'Privacy · Skill Registry',
      description: 'How the Skill Registry website handles site usage data and product interactions.',
      section: 'legal',
    };
  }

  if (pathname === '/terms') {
    return {
      title: 'Terms · Skill Registry',
      description: 'Terms for using the Skill Registry website and related materials.',
      section: 'legal',
    };
  }

  return {
    title: 'Page not found · Skill Registry',
    description: 'The requested page is not available. Return home or read the quick start.',
    section: 'not-found',
  };
}
