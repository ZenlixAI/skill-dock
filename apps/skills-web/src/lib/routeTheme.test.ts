import { describe, expect, it } from 'vitest';
import { getRouteTheme } from './routeTheme';

describe('getRouteTheme', () => {
  it('marks overview and catalog as frontstage routes', () => {
    expect(getRouteTheme('/').tone).toBe('frontstage');
    expect(getRouteTheme('/catalog').tone).toBe('frontstage');
  });

  it('marks workbench home as a backstage detail route', () => {
    expect(getRouteTheme('/workbench')).toEqual({
      tone: 'backstage',
      accent: 'detail',
    });
  });

  it('marks workbench import as a backstage console route', () => {
    expect(getRouteTheme('/workbench/import')).toEqual({
      tone: 'backstage',
      accent: 'console',
    });
  });

  it('marks other workbench routes as backstage console routes', () => {
    expect(getRouteTheme('/workbench/audit')).toEqual({
      tone: 'backstage',
      accent: 'console',
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

  it('marks login as a backstage console route', () => {
    expect(getRouteTheme('/login')).toEqual({
      tone: 'backstage',
      accent: 'console',
    });
  });

  it('treats skill detail routes as frontstage detail routes', () => {
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill').tone).toBe('frontstage');
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0').accent).toBe('detail');
  });
});
