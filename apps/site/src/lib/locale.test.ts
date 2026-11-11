import { describe, expect, it } from 'vitest';
import { localeFromPathname, localizePath, stripLocalePrefix } from './locale';

describe('locale routing', () => {
  it('detects the locale from a pathname', () => {
    expect(localeFromPathname('/')).toBe('en');
    expect(localeFromPathname('/registry')).toBe('en');
    expect(localeFromPathname('/zh')).toBe('zh');
    expect(localeFromPathname('/zh/')).toBe('zh');
    expect(localeFromPathname('/zh/registry')).toBe('zh');
  });

  it('strips the Chinese prefix without losing the rest of the path', () => {
    expect(stripLocalePrefix('/zh')).toBe('/');
    expect(stripLocalePrefix('/zh/registry')).toBe('/registry');
    expect(stripLocalePrefix('/zh/registry/')).toBe('/registry');
    expect(stripLocalePrefix('/registry/')).toBe('/registry');
    expect(stripLocalePrefix('/registry')).toBe('/registry');
  });

  it('localizes paths while preserving queries and hashes', () => {
    expect(localizePath('/', 'zh')).toBe('/zh');
    expect(localizePath('/', 'en')).toBe('/');
    expect(localizePath('/registry?view=trending#results', 'zh')).toBe(
      '/zh/registry?view=trending#results'
    );
    expect(localizePath('/#quick-start', 'zh')).toBe('/zh#quick-start');
    expect(localizePath('/zh/privacy', 'en')).toBe('/privacy');
    expect(localizePath('/zh/privacy', 'zh')).toBe('/zh/privacy');
    expect(localizePath('#faq', 'zh')).toBe('#faq');
    expect(localizePath('https://example.com/docs', 'zh')).toBe('https://example.com/docs');
  });
});
