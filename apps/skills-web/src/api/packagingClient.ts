import { requestJSON } from './httpClient';
import type {
  PackagingImportReviewRequest,
  PackagingImportReviewResponse,
  PackagingPublishRequest,
  PackagingPublishResponse,
  RegistryApiClient,
} from './types';

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
  'importReviewPackaging' | 'publishSkill'
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
};
