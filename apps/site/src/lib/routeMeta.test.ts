import { describe, expect, it } from 'vitest';
import { getRouteMeta } from './routeMeta';

describe('getRouteMeta', () => {
  it('returns portal metadata for the home page', () => {
    expect(getRouteMeta('/')).toEqual({
      title: 'Skill Registry · official website',
      description: 'Discover the product, read the docs, and follow the release path that keeps the site and backend aligned.',
      section: 'portal',
    });
  });

  it('returns portal metadata for the features page', () => {
    expect(getRouteMeta('/features')).toEqual({
      title: 'Features · Skill Registry',
      description: 'See how Skill Registry keeps the website focused, the backend self-hosted, and the deployment path clear.',
      section: 'portal',
    });
  });

  it('returns portal metadata for the docs page', () => {
    expect(getRouteMeta('/docs')).toEqual({
      title: 'Get started · Skill Registry',
      description: 'Deploy the backend, set the environment variables, and point the CLI at the same service.',
      section: 'portal',
    });
  });

  it('returns legal metadata for the privacy and terms pages', () => {
    expect(getRouteMeta('/privacy')).toEqual({
      title: 'Privacy · Skill Registry',
      description: 'Understand how Skill Registry handles site usage data and product interactions.',
      section: 'legal',
    });
    expect(getRouteMeta('/terms')).toEqual({
      title: 'Terms · Skill Registry',
      description: 'Review the terms for using the Skill Registry site and related materials.',
      section: 'legal',
    });
  });

  it('returns not-found metadata for unknown routes', () => {
    expect(getRouteMeta('/catalog')).toEqual({
      title: 'Page not found · Skill Registry',
      description: 'The requested page is not available. Return home or read the product features.',
      section: 'not-found',
    });
  });
});
