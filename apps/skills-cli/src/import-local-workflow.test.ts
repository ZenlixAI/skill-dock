import { spawnSync } from 'child_process';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('child_process', () => ({
  spawnSync: vi.fn(),
}));

vi.mock('./source-archive.ts', () => ({
  createSourceArchiveBase64: vi.fn(() => 'archive-base64'),
}));

import { runImportLocalWorkflow } from './import-local-workflow.ts';
import { createSourceArchiveBase64 } from './source-archive.ts';

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
    vi.mocked(spawnSync).mockReset();
    vi.mocked(createSourceArchiveBase64).mockClear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
    delete process.env.NPM_CONFIG_USERCONFIG;
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('uploads source archive content and completes the import flow', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );
    mkdirSync(join(testDir, 'docs'), { recursive: true });
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            warnings: ['normalized_missing_package_json'],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ok: true,
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            bundlePath: '/tmp/workspace/legacy-skill-0.1.0.tgz',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            bundleBase64: Buffer.from('tgz-data').toString('base64'),
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(null, {
          status: 404,
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        })
      );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      npmRegistry: 'https://npm.example.com',
    });

    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      bundlePath: expect.stringContaining('legacy-skill-0.1.0.tgz'),
      warnings: ['normalized_missing_package_json'],
    });
    const preparePayload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(preparePayload.source).toEqual({
      type: 'upload_archive',
      archiveBase64: 'archive-base64',
    });
    expect(preparePayload.options).toEqual({});
    const releaseRecordPayload = JSON.parse(String(fetchMock.mock.calls[4]?.[1]?.body));
    expect(releaseRecordPayload).toEqual({
      manifestDigest: 'sha256:test',
      components: ['docs'],
    });
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      'https://catalog.example.com/api/v1/skills/%40acme.skill%2Flegacy-skill/versions/0.1.0/release-record',
      expect.objectContaining({ method: 'PUT' })
    );
  });

  it('preserves an injected npm userconfig for publish during import-local', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );
    process.env.NPM_CONFIG_USERCONFIG = '/tmp/external-import.npmrc';
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ok: true,
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            bundlePath: '/tmp/workspace/legacy-skill-0.1.0.tgz',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            bundleBase64: Buffer.from('tgz-data').toString('base64'),
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(null, {
          status: 404,
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        })
      );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      npmRegistry: 'https://npm.example.com',
    });

    expect(result.ok).toBe(true);
    const publishCall = vi.mocked(spawnSync).mock.calls[0];
    expect(publishCall?.[2]).toMatchObject({
      env: expect.objectContaining({
        NPM_CONFIG_USERCONFIG: '/tmp/external-import.npmrc',
      }),
    });
  });

  it('skips npm publish when the imported version already exists in the npm registry and still writes the release record', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ok: true,
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            bundlePath: '/tmp/workspace/legacy-skill-0.1.0.tgz',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            bundleBase64: Buffer.from('tgz-data').toString('base64'),
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            name: '@acme.skill/legacy-skill',
            versions: {
              '0.1.0': {
                name: '@acme.skill/legacy-skill',
                version: '0.1.0',
              },
            },
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        })
      );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      npmRegistry: 'https://npm.example.com',
    });

    expect(spawnSync).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(5);
    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      'https://npm.example.com/%40acme.skill%2Flegacy-skill',
      expect.objectContaining({ method: 'GET' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      'https://catalog.example.com/api/v1/skills/%40acme.skill%2Flegacy-skill/versions/0.1.0/release-record',
      expect.objectContaining({ method: 'PUT' })
    );
    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      bundlePath: expect.stringContaining('legacy-skill-0.1.0.tgz'),
    });
  });

  it('recovers import-local when npm reports the version is already published but the release record is still missing', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 1,
      stdout: '',
      stderr: 'npm error You cannot publish over the previously published versions: 0.1.0.',
    } as ReturnType<typeof spawnSync>);

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ok: true,
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            bundlePath: '/tmp/workspace/legacy-skill-0.1.0.tgz',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            bundleBase64: Buffer.from('tgz-data').toString('base64'),
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            name: '@acme.skill/legacy-skill',
            versions: {},
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            name: '@acme.skill/legacy-skill',
            versions: {
              '0.1.0': {
                name: '@acme.skill/legacy-skill',
                version: '0.1.0',
              },
            },
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        })
      );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      npmRegistry: 'https://npm.example.com',
    });

    expect(spawnSync).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(6);
    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      'https://npm.example.com/%40acme.skill%2Flegacy-skill',
      expect.objectContaining({ method: 'GET' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      'https://npm.example.com/%40acme.skill%2Flegacy-skill',
      expect.objectContaining({ method: 'GET' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      6,
      'https://catalog.example.com/api/v1/skills/%40acme.skill%2Flegacy-skill/versions/0.1.0/release-record',
      expect.objectContaining({ method: 'PUT' })
    );
    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      bundlePath: expect.stringContaining('legacy-skill-0.1.0.tgz'),
    });
  });

  it('normalizes an explicit publishRegistry before prepare and release-record writes', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ok: true,
            workspacePath: '/tmp/workspace',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            bundlePath: '/tmp/workspace/legacy-skill-0.1.0.tgz',
            packageName: '@acme.skill/legacy-skill',
            version: '0.1.0',
            bundleBase64: Buffer.from('tgz-data').toString('base64'),
            manifestDigest: 'sha256:test',
            components: ['docs'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(null, {
          status: 404,
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        })
      );

    const result = await runImportLocalWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      publishRegistry: 'https://npm.example.com/',
      npmRegistry: 'https://npm.example.com/',
    });

    expect(result.ok).toBe(true);
    const preparePayload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(preparePayload.options).toEqual({
      publishRegistry: 'https://npm.example.com',
    });
    const releaseRecordPayload = JSON.parse(String(fetchMock.mock.calls[4]?.[1]?.body));
    expect(releaseRecordPayload).toEqual({
      publishRegistry: 'https://npm.example.com',
      manifestDigest: 'sha256:test',
      components: ['docs'],
    });
    const publishCall = vi.mocked(spawnSync).mock.calls[0];
    expect(publishCall?.[1]).toEqual([
      'publish',
      expect.stringContaining('legacy-skill-0.1.0.tgz'),
      '--registry',
      'https://npm.example.com',
    ]);
  });

  it('returns an actionable hint when catalog cannot create the packaging workspace', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );

    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          error: {
            code: 'internal_error',
            message: 'Failed to create packaging workspace.',
          },
        }),
        { status: 500, headers: { 'content-type': 'application/json' } }
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
        'Failed to create packaging workspace. Catalog write-side packaging workspace root is missing or not writable; check SKILL_REGISTRY_SHARED_WORKSPACE_ROOT and its mounted directory.',
      ],
    });
  });
});
