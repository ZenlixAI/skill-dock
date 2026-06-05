import { describe, expect, it } from 'vitest';
import {
  buildCatalogMutationResult,
  createMutationState,
} from './useCatalogMutation';

describe('catalog mutation helpers', () => {
  it('creates an empty mutation state', () => {
    expect(createMutationState()).toEqual({
      loading: false,
      error: null,
      success: null,
    });
  });

  it('builds a success result for release-record writes', () => {
    expect(
      buildCatalogMutationResult('release-record', 'succeeded', '已创建新的 release record。', {
        requestId: undefined,
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        actor: 'svc-release',
      })
    ).toEqual({
      action: 'release-record',
      outcome: 'succeeded',
      message: '已创建新的 release record。',
      requestId: null,
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      actor: 'svc-release',
    });
  });
});
