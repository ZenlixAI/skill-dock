import { existsSync, mkdirSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./source-archive.ts', () => ({
  createSourceArchiveBase64: vi.fn(() => 'archive-base64'),
}));

import { runImportLocalWorkflow } from './import-local-workflow.ts';

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

describe('import-local workflow', () => {
  let testDir: string;
  let originalFetch: typeof globalThis.fetch;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-import-local-workflow-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });
    originalFetch = globalThis.fetch;
    fetchMock = vi.fn();
    globalThis.fetch = fetchMock as unknown as typeof globalThis.fetch;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('uploads source archive content and completes stateless publish', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );
    mkdirSync(join(testDir, 'docs'), { recursive: true });

    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        packageName: '@acme.skill/legacy-skill',
        version: '0.1.0',
        manifestDigest: 'sha256:test',
        components: ['docs'],
        publishedToRegistry: true,
        created: true,
        idempotentReplay: false,
        versionStatus: 'published',
        installPolicy: 'resolvable',
        warnings: ['normalized_missing_package_json'],
        warningDetails: [
          {
            code: 'normalized_missing_package_json',
            severity: 'warning',
            title: 'Generated package.json',
            message: 'Catalog generated package.json during import.',
          },
        ],
        importReport: {
          sourceType: 'imported',
          strictMode: false,
          normalized: true,
          generatedPackageJson: true,
          generatedCapabilityPlaceholder: false,
          rewrotePackageIdentity: false,
          warningCodes: ['normalized_missing_package_json'],
        },
        errors: [],
      })
    );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
    });

    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      manifestDigest: 'sha256:test',
      components: ['docs'],
      warnings: ['normalized_missing_package_json'],
      warningDetails: [
        {
          code: 'normalized_missing_package_json',
          severity: 'warning',
          title: 'Generated package.json',
          message: 'Catalog generated package.json during import.',
        },
      ],
      importReport: {
        sourceType: 'imported',
        strictMode: false,
        normalized: true,
        generatedPackageJson: true,
        generatedCapabilityPlaceholder: false,
        rewrotePackageIdentity: false,
        warningCodes: ['normalized_missing_package_json'],
      },
      publishedToRegistry: true,
      created: true,
      idempotentReplay: false,
      versionStatus: 'published',
      installPolicy: 'resolvable',
    });

    expect(fetchMock).toHaveBeenCalledWith(
      'https://catalog.example.com/api/v1/skill-packaging/publish',
      expect.objectContaining({ method: 'POST' })
    );

    const payload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(payload.source).toEqual({
      type: 'upload_archive',
      archiveBase64: 'archive-base64',
    });
    expect(payload.options).toEqual({});
  });

  it('accepts null arrays from stateless publish responses', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );

    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        packageName: '@acme.skill/legacy-skill',
        version: '0.1.0',
        manifestDigest: 'sha256:test',
        components: null,
        publishedToRegistry: true,
        created: true,
        idempotentReplay: false,
        versionStatus: 'published',
        installPolicy: 'resolvable',
        warnings: null,
        warningDetails: null,
        importReport: null,
        errors: null,
      })
    );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
    });

    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      manifestDigest: 'sha256:test',
      components: [],
      publishedToRegistry: true,
      created: true,
      idempotentReplay: false,
      versionStatus: 'published',
      installPolicy: 'resolvable',
    });
  });

  it('threads strict and publishRegistry options to stateless publish', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify(
        {
          skill: {
            sourceType: 'platform_created',
          },
          publishConfig: {
            registry: 'https://npm.example.com',
          },
        },
        null,
        2
      )
    );

    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        packageName: '@acme.skill/legacy-skill',
        version: '0.1.0',
        manifestDigest: 'sha256:test',
        components: ['docs'],
        publishedToRegistry: true,
        created: true,
        idempotentReplay: false,
        versionStatus: 'published',
        installPolicy: 'resolvable',
        warnings: [],
        errors: [],
      })
    );

    await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      strict: true,
    });

    const payload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(payload.metadata).toEqual({
      displayName: 'legacy-skill',
      description: 'Imported skill',
      sourceType: 'platform_created',
    });
    expect(payload).not.toHaveProperty('includeBundleBase64');
    expect(payload.options).toEqual({
      publishRegistry: 'https://npm.example.com',
      strict: true,
    });
  });

  it('returns publish errors from catalog', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );

    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        packageName: '@acme.skill/legacy-skill',
        version: '0.1.0',
        manifestDigest: 'sha256:test',
        components: [],
        publishedToRegistry: false,
        created: false,
        idempotentReplay: false,
        versionStatus: 'published',
        installPolicy: 'blocked',
        warnings: [],
        errors: ['manifest_digest_mismatch'],
      })
    );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
    });

    expect(result).toEqual({
      ok: false,
      errors: ['manifest_digest_mismatch'],
    });
  });

  it('includes Catalog error context in publish failures', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );

    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        {
          error: {
            code: 'immutable_release_conflict',
            message: 'Immutable release content conflicts with the existing record.',
            details: {
              packageName: '@acme.skill/legacy-skill',
              version: '0.1.0',
            },
            requestId: 'req_123',
          },
        },
        409
      )
    );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
    });

    expect(result).toEqual({
      ok: false,
      errors: [
        'Immutable release content conflicts with the existing record. (catalog_status=409, catalog_code=immutable_release_conflict, catalog_request_id=req_123, catalog_details={"packageName":"@acme.skill/legacy-skill","version":"0.1.0"})',
      ],
    });
  });

  it('keeps Catalog error context with workspace remediation advice', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );

    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        {
          error: {
            code: 'internal_error',
            message: 'Failed to create packaging workspace.',
            details: { root: '/var/lib/skill-registry/packaging-workspaces' },
            requestId: 'req_456',
          },
        },
        500
      )
    );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
    });

    expect(result).toEqual({
      ok: false,
      errors: [
        'Failed to create packaging workspace. (catalog_status=500, catalog_code=internal_error, catalog_request_id=req_456, catalog_details={"root":"/var/lib/skill-registry/packaging-workspaces"}) Catalog write-side packaging workspace root is missing or not writable; check SKILL_REGISTRY_SHARED_WORKSPACE_ROOT and its mounted directory.',
      ],
    });
  });

  it('fails validation when package identity overrides are still missing', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      errors: [
        'packageName is required; pass --package-name when package.json is missing',
        'version is required; pass --version when package.json is missing',
      ],
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
