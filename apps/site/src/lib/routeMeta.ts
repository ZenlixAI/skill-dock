export type RouteSection = 'portal' | 'legal' | 'not-found';

export interface RouteMeta {
  title: string;
  description: string;
  section: RouteSection;
}

export function getRouteMeta(pathname: string): RouteMeta {
  if (pathname === '/') {
    return {
      title: 'Skill Dock · A self-hosted registry for agent skills',
      description:
        'Skill Dock is a self-hosted registry for agent skills. Store bundles in your npm registry, publish exact versions, resolve installs, and govern releases.',
      section: 'portal',
    };
  }

  if (pathname === '/privacy') {
    return {
      title: 'Privacy · Skill Dock',
      description: 'How the Skill Dock website handles site usage data.',
      section: 'legal',
    };
  }

  if (pathname === '/terms') {
    return {
      title: 'Terms · Skill Dock',
      description: 'Terms for using the Skill Dock website and related materials.',
      section: 'legal',
    };
  }

  return {
    title: 'Page not found · Skill Dock',
    description: 'The requested page is not available. Return home or read the quick start.',
    section: 'not-found',
  };
}
