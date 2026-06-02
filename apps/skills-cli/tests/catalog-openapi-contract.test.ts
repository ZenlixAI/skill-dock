import { readFileSync } from 'fs';
import { join } from 'path';
import { parse } from 'yaml';
import { describe, expect, it } from 'vitest';

type OpenAPISpec = {
  openapi: string;
  servers?: Array<{ url?: string }>;
  paths?: Record<string, Record<string, any>>;
  components?: {
    schemas?: Record<string, any>;
  };
};

function loadOpenAPISpec(): OpenAPISpec {
  const path = join(
    import.meta.dirname,
    '..',
    '..',
    '..',
    'docs',
    'superpowers',
    'specs',
    '2026-05-28-agent-skill-registry-catalog-api.openapi.yaml'
  );
  return parse(readFileSync(path, 'utf-8')) as OpenAPISpec;
}

function parameterNames(operation: Record<string, any>): string[] {
  return (operation.parameters || []).map((parameter: any) => {
    const ref = parameter.$ref as string | undefined;
    return ref ? ref.split('/').at(-1) : parameter.name;
  });
}

function responseSchemaRef(operation: Record<string, any>, status: string): string | undefined {
  return operation.responses?.[status]?.content?.['application/json']?.schema?.$ref;
}

describe('catalog OpenAPI contract', () => {
  const spec = loadOpenAPISpec();
  const paths = spec.paths || {};
  const schemas = spec.components?.schemas || {};

  it('documents exactly the catalog route surface and excludes packaging routes', () => {
    expect(spec.openapi).toBe('3.1.0');
    expect(spec.servers?.[0]?.url).toBe('/api/v1');

    expect(Object.keys(paths).sort()).toEqual(
      [
        '/audit-events',
        '/governance-changes',
        '/skills',
        '/skills/search',
        '/skills/curated',
        '/skills/{packageName}',
        '/skills/{packageName}/governance-changes',
        '/skills/{packageName}/install-target',
        '/skills/{packageName}/versions',
        '/skills/{packageName}/versions/{version}',
        '/skills/{packageName}/versions/{version}/governance-changes',
        '/skills/{packageName}/versions/{version}/release-record',
      ].sort()
    );

    expect(paths['/skill-packaging/prepare']).toBeUndefined();
    expect(paths['/skill-packaging/validate']).toBeUndefined();
    expect(paths['/skill-packaging/pack']).toBeUndefined();

    expect(Object.keys(paths['/skills'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/skills/search'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/skills/curated'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/skills/{packageName}'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/skills/{packageName}/versions'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/skills/{packageName}/versions/{version}'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/skills/{packageName}/install-target'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/governance-changes'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/audit-events'] || {})).toEqual(['get']);
    expect(Object.keys(paths['/skills/{packageName}/governance-changes'] || {})).toEqual(['post']);
    expect(Object.keys(paths['/skills/{packageName}/versions/{version}/governance-changes'] || {})).toEqual([
      'post',
    ]);
    expect(Object.keys(paths['/skills/{packageName}/versions/{version}/release-record'] || {})).toEqual([
      'put',
    ]);
  });

  it('pins key catalog operations to the expected params, roles, and response schemas', () => {
    const listSkills = paths['/skills']?.get;
    expect(listSkills?.['x-required-roles']).toEqual(['reader', 'publisher']);
    expect(parameterNames(listSkills)).toEqual([
      'Q',
      'PackageNameQuery',
      'SourceTypeQuery',
      'IncludeRetired',
      'PageSize',
      'Cursor',
      'SkillsSort',
    ]);
    expect(responseSchemaRef(listSkills, '200')).toBe('#/components/schemas/SkillsListResponse');

    const installTarget = paths['/skills/{packageName}/install-target']?.get;
    expect(installTarget?.['x-required-roles']).toEqual(['reader', 'publisher']);
    expect(parameterNames(installTarget)).toEqual(['PackageNamePath', 'VersionQuery']);
    expect(responseSchemaRef(installTarget, '200')).toBe(
      '#/components/schemas/InstallResolutionResponse'
    );
    expect(Object.keys(installTarget?.responses || {}).sort()).toEqual([
      '200',
      '401',
      '403',
      '404',
      '409',
      '422',
    ]);

    const releaseRecord = paths['/skills/{packageName}/versions/{version}/release-record']?.put;
    expect(releaseRecord?.['x-required-roles']).toEqual(['publisher']);
    expect(parameterNames(releaseRecord)).toEqual(['PackageNamePath', 'VersionPath']);
    expect(responseSchemaRef(releaseRecord, '200')).toBe(
      '#/components/schemas/ReleaseRecordResponse'
    );
    expect(responseSchemaRef(releaseRecord, '201')).toBe(
      '#/components/schemas/ReleaseRecordResponse'
    );
    expect(Object.keys(releaseRecord?.responses || {}).sort()).toEqual([
      '200',
      '201',
      '401',
      '403',
      '404',
      '409',
      '422',
      '503',
    ]);

    const skillGovernance = paths['/skills/{packageName}/governance-changes']?.post;
    expect(skillGovernance?.['x-required-roles']).toEqual(['publisher']);
    expect(skillGovernance.requestBody?.content?.['application/json']?.schema?.$ref).toBe(
      '#/components/schemas/SkillGovernanceChangeRequest'
    );

    const versionGovernance =
      paths['/skills/{packageName}/versions/{version}/governance-changes']?.post;
    expect(versionGovernance?.['x-required-roles']).toEqual(['publisher']);
    expect(versionGovernance.requestBody?.content?.['application/json']?.schema?.$ref).toBe(
      '#/components/schemas/VersionGovernanceChangeRequest'
    );
  });

  it('keeps core shared schemas aligned with the retained V1 contract', () => {
    expect(schemas.ErrorResponse?.required).toEqual(['error']);
    expect(schemas.PageInfo?.required).toEqual(['nextCursor', 'hasNextPage']);
    expect(schemas.SkillSummary?.required).toEqual([
      'packageName',
      'displayName',
      'description',
      'sourceType',
      'status',
      'latestVersion',
      'latestVersionStatus',
      'updatedAt',
    ]);
    expect(schemas.InstallResolutionResponse?.required).toEqual([
      'packageName',
      'version',
      'npmRegistry',
      'integrity',
      'tarballUrl',
      'skillManifestDigest',
      'versionStatus',
    ]);
    expect(schemas.ReleaseRecordResponse?.required).toEqual([
      'packageName',
      'version',
      'created',
      'idempotentReplay',
      'versionStatus',
      'installPolicy',
    ]);
    expect(schemas.SkillGovernanceChangeRequest?.properties?.action?.enum).toEqual(['retire']);
    expect(schemas.VersionGovernanceChangeRequest?.properties?.action?.enum).toEqual([
      'deprecate',
      'yank',
    ]);
  });
});
