import { describe, expect, it } from 'vitest';
import { getRouteMeta } from './routeMeta';

describe('getRouteMeta', () => {
  it('returns product metadata for the home page', () => {
    expect(getRouteMeta('/')).toEqual({
      title: 'Skill Registry · A private catalog for agent skills',
      description:
        'A self-hosted catalog for agent skills. Store bundles in your npm registry, publish exact versions, resolve installs, and govern releases.',
      section: 'portal',
    });
  });

  it('returns legal metadata for the privacy and terms pages', () => {
    expect(getRouteMeta('/privacy')).toEqual({
      title: 'Privacy · Skill Registry',
      description: 'How the Skill Registry website handles site usage data and product interactions.',
      section: 'legal',
    });
    expect(getRouteMeta('/terms')).toEqual({
      title: 'Terms · Skill Registry',
      description: 'Terms for using the Skill Registry website and related materials.',
      section: 'legal',
    });
  });

  it('returns not-found metadata for unknown routes', () => {
    expect(getRouteMeta('/catalog')).toEqual({
      title: 'Page not found · Skill Registry',
      description: 'The requested page is not available. Return home or read the quick start.',
      section: 'not-found',
    });
  });
});
