import type { RegistryClientConfig } from './types';

interface RegistryRuntimeEnv {
  VITE_SKILL_REGISTRY_URL?: string;
  VITE_SKILL_REGISTRY_USE_MOCK?: string;
  VITE_SKILL_REGISTRY_ROLE?: string;
  VITE_SKILL_REGISTRY_ACTOR?: string;
  VITE_SKILL_REGISTRY_TOKEN?: string;
}

function normalizeBoolean(value: string | undefined) {
  if (!value) {
    return false;
  }
  return ['1', 'true', 'yes', 'on'].includes(value.trim().toLowerCase());
}

function normalizeBaseUrl(value: string | undefined) {
  const trimmed = value?.trim();
  if (!trimmed) {
    return null;
  }
  return trimmed.replace(/\/+$/, '');
}

export function deriveRegistryClientConfig(env: RegistryRuntimeEnv): RegistryClientConfig {
  const baseUrl = normalizeBaseUrl(env.VITE_SKILL_REGISTRY_URL);
  const forceMock = normalizeBoolean(env.VITE_SKILL_REGISTRY_USE_MOCK);

  return {
    mode: forceMock ? 'mock' : 'real',
    baseUrl,
    role: env.VITE_SKILL_REGISTRY_ROLE?.trim() || null,
    actor: env.VITE_SKILL_REGISTRY_ACTOR?.trim() || null,
    hasToken: Boolean(env.VITE_SKILL_REGISTRY_TOKEN?.trim()),
  };
}

export const registryClientConfig: RegistryClientConfig = deriveRegistryClientConfig(
  import.meta.env as RegistryRuntimeEnv
);

export function getRegistryToken() {
  return import.meta.env.VITE_SKILL_REGISTRY_TOKEN?.trim() || null;
}
