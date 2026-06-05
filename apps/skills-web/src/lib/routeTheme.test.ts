import { describe, expect, it } from 'vitest';
import { getRouteTheme } from './routeTheme';

describe('getRouteTheme', () => {
  it('marks overview and catalog as frontstage routes', () => {
    expect(getRouteTheme('/').tone).toBe('frontstage');
    expect(getRouteTheme('/catalog').tone).toBe('frontstage');
  });

  it('marks workbench routes as backstage routes', () => {
    expect(getRouteTheme('/workbench').tone).toBe('backstage');
    expect(getRouteTheme('/workbench/import').tone).toBe('backstage');
    expect(getRouteTheme('/workbench/audit').tone).toBe('backstage');
  });

  it('treats skill detail routes as frontstage detail routes', () => {
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill').tone).toBe('frontstage');
    expect(getRouteTheme('/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0').accent).toBe('detail');
  });
});
