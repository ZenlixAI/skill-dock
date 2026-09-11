import { describe, expect, it } from 'vitest';
import { getRouteMeta } from './routeMeta';

describe('getRouteMeta', () => {
  it('returns portal metadata for the home page', () => {
    expect(getRouteMeta('/')).toEqual({
      title: 'Skill Registry · official website',
      description:
        'A self-hosted registry for discovering, versioning, and governing the AI agent skills your team relies on.',
      section: 'portal',
    });
  });

  it('returns portal metadata for the features page', () => {
    expect(getRouteMeta('/features')).toEqual({
      title: 'Features · Skill Registry',
      description:
        'A private package registry for installing and sharing reusable agent skills.',
      section: 'portal',
    });
  });

  it('returns not-found metadata for removed installation routes', () => {
    expect(getRouteMeta('/docs')).toEqual({
      title: 'Page not found · Skill Registry',
      description: 'The requested page is not available. Return home or read the product features.',
      section: 'not-found',
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
