import { registryClientConfig } from './config';
import { RegistryHttpError, requestJSON } from './httpClient';
import type { RegistryApiClient, RegistrySession } from './types';

export const sessionClient: Pick<
  RegistryApiClient,
  'getConfig' | 'getSession' | 'loginSession' | 'logoutSession'
> = {
  getConfig() {
    return registryClientConfig;
  },

  getSession() {
    return requestJSON<RegistrySession | null>('/api/v1/session').catch((error: unknown) => {
      if (error instanceof RegistryHttpError && error.status === 401) {
        return null;
      }
      throw error;
    });
  },

  loginSession(username: string, password: string) {
    return requestJSON<RegistrySession>('/api/v1/session/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  },

  async logoutSession() {
    await requestJSON<{ ok: boolean }>('/api/v1/session/logout', {
      method: 'POST',
      body: JSON.stringify({}),
    });
  },
};
