import { requestJSON } from './httpClient';
import type {
  PackagingImportReviewRequest,
  PackagingImportReviewResponse,
  PackagingCleanupRequest,
  PackagingCleanupResponse,
  PackagingPackRequest,
  PackagingPackResponse,
  PackagingPublishRequest,
  PackagingPublishResponse,
  PackagingPrepareRequest,
  PackagingPrepareResponse,
  PackagingValidateRequest,
  PackagingValidateResponse,
  RegistryApiClient,
} from './types';

function withQuery(path: string, params: Record<string, string | number | boolean | undefined>) {
  const url = new URL(path, 'http://registry.local');
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === '' || value === false) {
      return;
    }
    url.searchParams.set(key, String(value));
  });
  return `${url.pathname}${url.search}`;
}

export function buildPackagingBundleDownloadPath(
  workspacePath: string,
  packageName: string,
  version: string
) {
  return withQuery('/api/v1/skill-packaging/bundle', {
    workspacePath,
    packageName,
    version,
  });
}

export function normalizePackagingReview<
  T extends { warnings?: string[] | null; warningDetails?: unknown[] | null; errors?: string[] | null },
>(response: T) {
  return {
    ...response,
    warnings: Array.isArray(response.warnings) ? response.warnings : [],
    warningDetails: Array.isArray(response.warningDetails) ? response.warningDetails : [],
    errors: Array.isArray(response.errors) ? response.errors : [],
  };
}

export const packagingClient: Pick<
  RegistryApiClient,
  | 'importReviewPackaging'
  | 'preparePackaging'
  | 'validatePackaging'
  | 'packSkill'
  | 'publishSkill'
  | 'cleanupPackagingWorkspace'
> = {
  async importReviewPackaging(request: PackagingImportReviewRequest) {
    const response = await requestJSON<PackagingImportReviewResponse>(
      '/api/v1/skill-packaging/import-review',
      {
        method: 'POST',
        body: JSON.stringify(request),
      }
    );
    return {
      ...normalizePackagingReview(response),
      components: Array.isArray(response.components) ? response.components : [],
    };
  },

  async preparePackaging(request: PackagingPrepareRequest) {
    const response = await requestJSON<PackagingPrepareResponse>('/api/v1/skill-packaging/prepare', {
      method: 'POST',
      body: JSON.stringify(request),
    });
    return normalizePackagingReview(response);
  },

  async validatePackaging(request: PackagingValidateRequest) {
    const response = await requestJSON<PackagingValidateResponse>(
      '/api/v1/skill-packaging/validate',
      {
        method: 'POST',
        body: JSON.stringify(request),
      }
    );
    return {
      ...normalizePackagingReview(response),
      components: Array.isArray(response.components) ? response.components : [],
    };
  },

  async packSkill(request: PackagingPackRequest) {
    const response = await requestJSON<PackagingPackResponse>('/api/v1/skill-packaging/pack', {
      method: 'POST',
      body: JSON.stringify(request),
    });
    return {
      ...normalizePackagingReview(response),
      components: Array.isArray(response.components) ? response.components : [],
    };
  },

  async publishSkill(request: PackagingPublishRequest) {
    const response = await requestJSON<PackagingPublishResponse>('/api/v1/skill-packaging/publish', {
      method: 'POST',
      body: JSON.stringify(request),
    });
    return {
      ...normalizePackagingReview(response),
      components: Array.isArray(response.components) ? response.components : [],
    };
  },

  cleanupPackagingWorkspace(request: PackagingCleanupRequest) {
    return requestJSON<PackagingCleanupResponse>('/api/v1/skill-packaging/cleanup', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  },
};
