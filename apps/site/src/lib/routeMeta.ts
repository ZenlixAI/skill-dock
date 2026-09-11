export type RouteSection = 'portal' | 'legal' | 'not-found';

export interface RouteMeta {
  title: string;
  description: string;
  section: RouteSection;
}

export function getRouteMeta(pathname: string): RouteMeta {
  if (pathname === '/') {
    return {
      title: 'Skill Registry · official website',
      description:
        'A self-hosted registry for discovering, versioning, and governing the AI agent skills your team relies on.',
      section: 'portal',
    };
  }

  if (pathname === '/features') {
    return {
      title: 'Features · Skill Registry',
      description:
        'A private package registry for installing and sharing reusable agent skills.',
      section: 'portal',
    };
  }

  if (pathname === '/privacy') {
    return {
      title: 'Privacy · Skill Registry',
      description: 'Understand how Skill Registry handles site usage data and product interactions.',
      section: 'legal',
    };
  }

  if (pathname === '/terms') {
    return {
      title: 'Terms · Skill Registry',
      description: 'Review the terms for using the Skill Registry site and related materials.',
      section: 'legal',
    };
  }

  return {
    title: 'Page not found · Skill Registry',
    description: 'The requested page is not available. Return home or read the product features.',
    section: 'not-found',
  };
}
