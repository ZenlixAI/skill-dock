import { describe, expect, it } from 'vitest';
import { getRouteMeta } from './routeMeta';

describe('getRouteMeta', () => {
  it('returns product metadata for the home page', () => {
    expect(getRouteMeta('/')).toEqual({
      title: 'Skill Dock · A self-hosted registry for agent skills',
      description:
        'Skill Dock is a self-hosted registry for agent skills. Store bundles in your npm registry, publish exact versions, resolve installs, and govern releases.',
      section: 'portal',
    });
  });

  it('returns legal metadata for the privacy and terms pages', () => {
    expect(getRouteMeta('/privacy')).toEqual({
      title: 'Privacy · Skill Dock',
      description: 'How the Skill Dock website handles site usage data.',
      section: 'legal',
    });
    expect(getRouteMeta('/terms')).toEqual({
      title: 'Terms · Skill Dock',
      description: 'Terms for using the Skill Dock website and related materials.',
      section: 'legal',
    });
  });

  it('returns not-found metadata for unknown routes', () => {
    expect(getRouteMeta('/catalog')).toEqual({
      title: 'Page not found · Skill Dock',
      description: 'The requested page is not available. Return home or read the quick start.',
      section: 'not-found',
    });
  });
});
