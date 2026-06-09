export type RouteSection = 'portal' | 'console' | 'auth' | 'legal' | 'not-found';

export interface RouteMeta {
  title: string;
  description: string;
  section: RouteSection;
}

const skillDetailPattern = /^\/skills\/[^/]+$/;
const versionDetailPattern = /^\/skills\/[^/]+\/versions\/[^/]+$/;

export function getRouteMeta(pathname: string): RouteMeta {
  if (pathname === '/') {
    return {
      title: 'Skill Registry Portal',
      description: 'Search, evaluate, and install skills from the company registry.',
      section: 'portal',
    };
  }

  if (pathname === '/catalog') {
    return {
      title: 'Catalog · Skill Registry',
      description: 'Browse curated and searchable skills with install-ready metadata.',
      section: 'portal',
    };
  }

  if (skillDetailPattern.test(pathname)) {
    return {
      title: 'Skill Detail · Skill Registry',
      description: 'Review package details, provenance, and install information for a registry skill.',
      section: 'portal',
    };
  }

  if (versionDetailPattern.test(pathname)) {
    return {
      title: 'Version Detail · Skill Registry',
      description: 'Inspect release metadata and install a specific registry skill version.',
      section: 'portal',
    };
  }

  if (pathname === '/workbench') {
    return {
      title: 'Console · Skill Registry',
      description: 'Operate import, authoring, governance, and audit workflows.',
      section: 'console',
    };
  }

  if (pathname === '/workbench/import') {
    return {
      title: 'Import · Skill Registry',
      description: 'Prepare and publish new skills into the company registry.',
      section: 'console',
    };
  }

  if (pathname === '/workbench/authoring') {
    return {
      title: 'Authoring · Skill Registry',
      description: 'Create and refine skill package content before publishing.',
      section: 'console',
    };
  }

  if (pathname === '/workbench/governance') {
    return {
      title: 'Governance · Skill Registry',
      description: 'Apply governance policy and review operational changes across skills.',
      section: 'console',
    };
  }

  if (pathname === '/workbench/audit') {
    return {
      title: 'Audit · Skill Registry',
      description: 'Review activity history and operational evidence from the registry.',
      section: 'console',
    };
  }

  if (pathname === '/login') {
    return {
      title: 'Login · Skill Registry',
      description: 'Sign in to the internal skill registry.',
      section: 'auth',
    };
  }

  if (pathname === '/privacy') {
    return {
      title: 'Privacy · Skill Registry',
      description: 'How the internal registry handles operator and catalog activity data.',
      section: 'legal',
    };
  }

  if (pathname === '/terms') {
    return {
      title: 'Terms · Skill Registry',
      description: 'Rules for using the internal skill registry portal and console.',
      section: 'legal',
    };
  }

  return {
    title: 'Page Not Found · Skill Registry',
    description: 'The requested page does not exist. Return to the portal overview or catalog.',
    section: 'not-found',
  };
}
