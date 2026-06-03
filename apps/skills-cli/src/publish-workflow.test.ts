import { spawnSync } from 'child_process';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { dirname, join } from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('child_process', () => ({
  spawnSync: vi.fn(),
}));

vi.mock('./packaging-session.ts', () => ({
  runPackagingSession: vi.fn(),
}));

import { runPublishWorkflow } from './publish-workflow.ts';
import { runPackagingSession } from './packaging-session.ts';

describe('publish workflow', () => {
  let testDir: string;
  let originalFetch: typeof globalThis.fetch;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-publish-workflow-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });

    originalFetch = globalThis.fetch;
    fetchMock = vi.fn();
    globalThis.fetch = fetchMock as unknown as typeof globalThis.fetch;

    vi.mocked(spawnSync).mockReset();
    vi.mocked(runPackagingSession).mockReset();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('runs packaging session, pack, publish, and release-record in order', async () => {
    writeFileSync(join(testDir, '.npmrc'), '//npm.example.com/:_authToken=publish-token\n');
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);

    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      workspacePath: '/tmp/packaging/workspace',
      idempotencyKey: 'packaging-session-key',
      manifestDigest: 'sha256:validate',
      publishRegistry: 'https://npm.example.com',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: [],
      errors: [],
    });

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/packaging/workspace',
            bundlePath: '/tmp/packaging/workspace/.dist/platform.skill-my-skill-1.2.0.tgz',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            manifestDigest: 'sha256:packed',
            components: ['commands'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(Buffer.from('tgz-data'), {
          status: 200,
          headers: { 'content-type': 'application/octet-stream' },
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        })
      );

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      registry: 'https://catalog.example.com',
    });
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://catalog.example.com/api/v1/skill-packaging/pack',
      expect.objectContaining({ method: 'POST' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://catalog.example.com/api/v1/skill-packaging/bundle?workspacePath=%2Ftmp%2Fpackaging%2Fworkspace&packageName=%40platform.skill%2Fmy-skill&version=1.2.0',
      expect.objectContaining({ method: 'GET' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      'https://catalog.example.com/api/v1/skills/%40platform.skill%2Fmy-skill/versions/1.2.0/release-record',
      expect.objectContaining({ method: 'PUT' })
    );
    const packPayload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    const releaseRecordPayload = JSON.parse(String(fetchMock.mock.calls[2]?.[1]?.body));
    expect(packPayload).toMatchObject({
      idempotencyKey: 'packaging-session-key',
      workspacePath: '/tmp/packaging/workspace',
      target: {
        packageName: '@platform.skill/my-skill',
        version: '1.2.0',
      },
      includeBundle: false,
    });
    expect(releaseRecordPayload).toEqual({
      publishRegistry: 'https://npm.example.com',
      manifestDigest: 'sha256:packed',
      components: ['commands'],
    });
    expect(spawnSync).toHaveBeenCalledTimes(1);
    const publishCall = vi.mocked(spawnSync).mock.calls[0];
    expect(publishCall?.[0]).toBe('npm');
    expect(publishCall?.[1]).toEqual([
      'publish',
      expect.stringContaining('my-skill-1.2.0.tgz'),
    ]);
    expect(publishCall?.[2]).toMatchObject({
      cwd: testDir,
      env: expect.objectContaining({
        NPM_CONFIG_USERCONFIG: join(testDir, '.npmrc'),
      }),
    });
    expect(result).toEqual({
      ok: true,
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      bundlePath: publishCall?.[1]?.[1],
    });
    if (result.ok) {
      expect(existsSync(result.bundlePath)).toBe(true);
      rmSync(dirname(result.bundlePath), { recursive: true, force: true });
    }
  });

  it('downloads the packed bundle when pack returns metadata only', async () => {
    writeFileSync(join(testDir, '.npmrc'), '//npm.example.com/:_authToken=publish-token\n');
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);

    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      workspacePath: '/tmp/packaging/workspace',
      idempotencyKey: 'packaging-session-key',
      manifestDigest: 'sha256:validate',
      publishRegistry: 'https://npm.example.com',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: [],
      errors: [],
    });

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/packaging/workspace',
            bundlePath: '/tmp/packaging/workspace/.dist/platform.skill-my-skill-1.2.0.tgz',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            manifestDigest: 'sha256:packed',
            components: ['commands'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(Buffer.from('tgz-data'), {
          status: 200,
          headers: { 'content-type': 'application/octet-stream' },
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        })
      );

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://catalog.example.com/api/v1/skill-packaging/bundle?workspacePath=%2Ftmp%2Fpackaging%2Fworkspace&packageName=%40platform.skill%2Fmy-skill&version=1.2.0',
      expect.objectContaining({ method: 'GET' })
    );
    expect(result).toEqual({
      ok: true,
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      bundlePath: expect.stringContaining('my-skill-1.2.0.tgz'),
    });
    if (result.ok) {
      expect(existsSync(result.bundlePath)).toBe(true);
      rmSync(dirname(result.bundlePath), { recursive: true, force: true });
    }
  });

  it('preserves packaging session warnings on successful publish results', async () => {
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);

    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      workspacePath: '/tmp/packaging/workspace',
      idempotencyKey: 'packaging-session-key',
      manifestDigest: 'sha256:validate',
      publishRegistry: 'https://npm.example.com',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: [],
      errors: [],
      warnings: ['normalized_missing_capability_content'],
      warningDetails: [
        {
          code: 'normalized_missing_capability_content',
          severity: 'warning',
          title: 'Generated placeholder capability content',
          message: 'The imported skill did not contain publishable capability content.',
          suggestedAction: 'Replace the placeholder with real capability content.',
        },
      ],
      importReport: {
        sourceType: 'imported',
        strictMode: false,
        normalized: true,
        generatedPackageJson: false,
        generatedCapabilityPlaceholder: true,
        rewrotePackageIdentity: false,
        generatedPaths: ['docs/README.md'],
        warningCodes: ['normalized_missing_capability_content'],
      },
    });

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/packaging/workspace',
            bundlePath: '/tmp/packaging/workspace/platform.skill-my-skill-1.2.0.tgz',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            manifestDigest: 'sha256:packed',
            components: ['commands'],
            warnings: [],
            errors: [],
            bundleBase64: Buffer.from('tgz-data').toString('base64'),
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

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: true,
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      bundlePath: expect.stringContaining('my-skill-1.2.0.tgz'),
      warnings: ['normalized_missing_capability_content'],
      warningDetails: [
        {
          code: 'normalized_missing_capability_content',
          severity: 'warning',
          title: 'Generated placeholder capability content',
          message: 'The imported skill did not contain publishable capability content.',
          suggestedAction: 'Replace the placeholder with real capability content.',
        },
      ],
      importReport: {
        sourceType: 'imported',
        strictMode: false,
        normalized: true,
        generatedPackageJson: false,
        generatedCapabilityPlaceholder: true,
        rewrotePackageIdentity: false,
        generatedPaths: ['docs/README.md'],
        warningCodes: ['normalized_missing_capability_content'],
      },
    });

    if (result.ok) {
      expect(existsSync(result.bundlePath)).toBe(true);
      rmSync(dirname(result.bundlePath), { recursive: true, force: true });
    }
  });

  it('returns packaging session errors without calling pack or publish', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: false,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'broken-skill',
      description: 'Missing version',
      errors: ['package.json version is required'],
      components: [],
    });

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      errors: ['package.json version is required'],
    });
    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      registry: 'https://catalog.example.com',
    });
    expect(fetchMock).not.toHaveBeenCalled();
    expect(spawnSync).not.toHaveBeenCalled();
  });

  it('uses the default catalog registry when no override is provided', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: false,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'broken-skill',
      description: 'Missing version',
      errors: ['package.json version is required'],
      components: [],
    });

    await runPublishWorkflow({
      path: testDir,
    });

    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      registry: 'https://skills.sh',
    });
  });

  it('does not register the release record when npm publish fails', async () => {
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 1,
      stdout: '',
      stderr: 'publish denied',
    } as ReturnType<typeof spawnSync>);

    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      workspacePath: '/tmp/packaging/workspace',
      idempotencyKey: 'packaging-session-key',
      manifestDigest: 'sha256:validate',
      publishRegistry: 'https://npm.example.com',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: [],
      errors: [],
    });

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/packaging/workspace',
            bundlePath: '/tmp/packaging/workspace/platform.skill-my-skill-1.2.0.tgz',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            manifestDigest: 'sha256:packed',
            components: ['commands'],
            warnings: [],
            errors: [],
            bundleBase64: Buffer.from('tgz-data').toString('base64'),
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      );

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      errors: ['publish denied'],
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
