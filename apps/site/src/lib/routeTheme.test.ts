import { describe, expect, it } from 'vitest';
import { getRouteTheme } from './routeTheme';

describe('getRouteTheme', () => {
  it('marks the public site routes as frontstage routes', () => {
    expect(getRouteTheme('/')).toEqual({
      tone: 'frontstage',
      accent: 'landing',
    });
    expect(getRouteTheme('/features')).toEqual({
      tone: 'frontstage',
      accent: 'detail',
    });
  });

  it('marks legal routes as frontstage legal routes', () => {
    expect(getRouteTheme('/privacy')).toEqual({
      tone: 'frontstage',
      accent: 'legal',
    });
    expect(getRouteTheme('/terms')).toEqual({
      tone: 'frontstage',
      accent: 'legal',
    });
  });

  it('treats removed app-style routes as neutral public pages', () => {
    expect(getRouteTheme('/catalog')).toEqual({
      tone: 'frontstage',
      accent: 'neutral',
    });
    expect(getRouteTheme('/registry')).toEqual({
      tone: 'frontstage',
      accent: 'neutral',
    });
    expect(getRouteTheme('/governance')).toEqual({
      tone: 'frontstage',
      accent: 'neutral',
    });
    expect(getRouteTheme('/audit')).toEqual({
      tone: 'frontstage',
      accent: 'neutral',
    });
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill')).toEqual({
      tone: 'frontstage',
      accent: 'neutral',
    });
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0')).toEqual({
      tone: 'frontstage',
      accent: 'neutral',
    });
  });
});
