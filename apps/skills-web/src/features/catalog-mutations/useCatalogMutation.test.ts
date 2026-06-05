import { describe, expect, it } from 'vitest';
import {
  buildCatalogMutationResult,
  createMutationState,
  persistCatalogMutation,
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

  it('persists mutation result params with defaults and request id', () => {
    const result = persistCatalogMutation(
      new URLSearchParams('tab=governance'),
      {
        action: 'retire',
        outcome: 'failed',
        message: 'retire failed',
        requestId: 'req_123',
      },
      {
        packageName: '@platform.skill/pdf-skill',
        version: null,
        actor: 'svc-release',
      }
    );

    expect(result.toString()).toBe(
      'tab=governance&mutation=retire&mutationOutcome=failed&mutationMessage=retire+failed&mutationRequestId=req_123&mutationPackage=%40platform.skill%2Fpdf-skill&mutationActor=svc-release'
    );
  });
});
