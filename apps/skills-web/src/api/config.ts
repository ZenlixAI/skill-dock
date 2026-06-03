import type { RegistryClientConfig } from './types';

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

const baseUrl = normalizeBaseUrl(import.meta.env.VITE_SKILL_REGISTRY_URL);
const forceMock = normalizeBoolean(import.meta.env.VITE_SKILL_REGISTRY_USE_MOCK);

export const registryClientConfig: RegistryClientConfig = {
  mode: forceMock || !baseUrl ? 'mock' : 'real',
  baseUrl,
  role: import.meta.env.VITE_SKILL_REGISTRY_ROLE?.trim() || null,
  actor: import.meta.env.VITE_SKILL_REGISTRY_ACTOR?.trim() || null,
  hasToken: Boolean(import.meta.env.VITE_SKILL_REGISTRY_TOKEN?.trim()),
};

export function getRegistryToken() {
  return import.meta.env.VITE_SKILL_REGISTRY_TOKEN?.trim() || null;
}
