import { describe, expect, it } from 'vitest';
import { getRouteTheme } from './routeTheme';

describe('getRouteTheme', () => {
  it('marks overview and catalog as frontstage routes', () => {
    expect(getRouteTheme('/').tone).toBe('frontstage');
    expect(getRouteTheme('/catalog').tone).toBe('frontstage');
  });

  it('marks import and audit as backstage routes', () => {
    expect(getRouteTheme('/import').tone).toBe('backstage');
    expect(getRouteTheme('/audit').tone).toBe('backstage');
  });

  it('treats skill detail routes as frontstage detail routes', () => {
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill').tone).toBe('frontstage');
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0').accent).toBe('detail');
  });
});
