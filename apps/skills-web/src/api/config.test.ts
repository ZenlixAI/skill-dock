import { describe, expect, it } from 'vitest';
import { deriveRegistryClientConfig } from './config';

describe('deriveRegistryClientConfig', () => {
  it('stays in real mode when baseUrl is missing unless mock is forced', () => {
    expect(deriveRegistryClientConfig({ VITE_SKILL_REGISTRY_URL: '', VITE_SKILL_REGISTRY_USE_MOCK: '' }).mode).toBe(
      'real'
    );
  });

  it('enters real mode only when baseUrl is present and mock is not forced', () => {
    expect(
      deriveRegistryClientConfig({
        VITE_SKILL_REGISTRY_URL: 'http://127.0.0.1:18080/',
        VITE_SKILL_REGISTRY_USE_MOCK: 'false',
      }).mode
    ).toBe('real');
  });

  it('forces mock mode when explicitly requested even with a baseUrl', () => {
    expect(
      deriveRegistryClientConfig({
        VITE_SKILL_REGISTRY_URL: 'http://127.0.0.1:18080/',
        VITE_SKILL_REGISTRY_USE_MOCK: 'true',
      }).mode
    ).toBe('mock');
  });

  it('normalizes baseUrl and trims actor metadata', () => {
    expect(
      deriveRegistryClientConfig({
        VITE_SKILL_REGISTRY_URL: '  http://127.0.0.1:18080///  ',
        VITE_SKILL_REGISTRY_USE_MOCK: 'false',
        VITE_SKILL_REGISTRY_ROLE: ' reader ',
        VITE_SKILL_REGISTRY_ACTOR: ' svc-reader ',
        VITE_SKILL_REGISTRY_TOKEN: ' secret-token ',
      })
    ).toEqual({
      mode: 'real',
      baseUrl: 'http://127.0.0.1:18080',
      role: 'reader',
      actor: 'svc-reader',
      hasToken: true,
    });
  });
});
