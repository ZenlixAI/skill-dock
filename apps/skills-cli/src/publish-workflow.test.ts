import { spawnSync } from 'child_process';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('child_process', () => ({
  spawnSync: vi.fn(),
}));

vi.mock('./packaging-session.ts', () => ({
  runPackagingSession: vi.fn(),
}));

vi.mock('./local-packaging.ts', () => ({
  packLocalSkillDirectory: vi.fn(),
}));

import { runPublishWorkflow } from './publish-workflow.ts';
import { packLocalSkillDirectory } from './local-packaging.ts';
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
    vi.mocked(packLocalSkillDirectory).mockReset();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
    delete process.env.NPM_CONFIG_USERCONFIG;
    delete process.env.npm_config_registry;
    delete process.env.NPM_CONFIG_REGISTRY;
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('runs local packaging, publish, and release-record in order', async () => {
    writeFileSync(join(testDir, '.npmrc'), '//npm.example.com/:_authToken=publish-token\n');
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);

    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      publishRegistry: 'https://npm.example.com',
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: ['commands'],
      errors: [],
    });
    const bundlePath = join(testDir, 'packed.tgz');
    vi.mocked(packLocalSkillDirectory).mockResolvedValueOnce({
      bundlePath,
      manifestDigest: 'sha256:packed',
      cleanup: vi.fn(),
    });

    fetchMock
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

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      strict: undefined,
    });
    expect(packLocalSkillDirectory).toHaveBeenCalledWith({
      directory: testDir,
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://npm.example.com/%40acme.skill%2Fmy-skill',
      expect.objectContaining({ method: 'GET' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://catalog.example.com/api/v1/skills/%40acme.skill%2Fmy-skill/versions/1.2.0/release-record',
      expect.objectContaining({ method: 'PUT' })
    );
    const releaseRecordPayload = JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body));
    expect(releaseRecordPayload).toEqual({
      publishRegistry: 'https://npm.example.com',
      manifestDigest: 'sha256:packed',
      components: ['commands'],
    });
    expect(spawnSync).toHaveBeenCalledTimes(1);
    const publishCall = vi.mocked(spawnSync).mock.calls[0];
    expect(publishCall?.[0]).toBe('npm');
    expect(publishCall?.[1]).toEqual(['publish', bundlePath, '--registry', 'https://npm.example.com']);
    expect(publishCall?.[2]).toMatchObject({
      cwd: testDir,
      env: expect.objectContaining({
        NPM_CONFIG_USERCONFIG: join(testDir, '.npmrc'),
      }),
    });
    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      bundlePath,
    });
  });

  it('uses the default catalog registry when no override is provided', async () => {
    vi.stubEnv('SKILLS_REGISTRY_URL', '');
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      publishRegistry: 'https://npm.example.com',
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: ['commands'],
      errors: [],
    });
    vi.mocked(packLocalSkillDirectory).mockResolvedValueOnce({
      bundlePath: join(testDir, 'packed.tgz'),
      manifestDigest: 'sha256:packed',
      cleanup: vi.fn(),
    });
    fetchMock
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

    await runPublishWorkflow({
      path: testDir,
    });

    expect(runPackagingSession).toHaveBeenCalledWith({
      path: testDir,
      strict: undefined,
    });
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://skills.sh/api/v1/skills/%40acme.skill%2Fmy-skill/versions/1.2.0/release-record',
      expect.objectContaining({ method: 'PUT' })
    );
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
      strict: undefined,
    });
    expect(packLocalSkillDirectory).not.toHaveBeenCalled();
    expect(fetchMock).not.toHaveBeenCalled();
    expect(spawnSync).not.toHaveBeenCalled();
  });

  it('does not register the release record when npm publish fails', async () => {
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 1,
      stdout: '',
      stderr: 'publish denied',
    } as ReturnType<typeof spawnSync>);

    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      publishRegistry: 'https://npm.example.com',
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: ['commands'],
      errors: [],
    });
    vi.mocked(packLocalSkillDirectory).mockResolvedValueOnce({
      bundlePath: join(testDir, 'packed.tgz'),
      manifestDigest: 'sha256:packed',
      cleanup: vi.fn(),
    });
    fetchMock.mockResolvedValueOnce(
      new Response(null, {
        status: 404,
      })
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

  it('skips npm publish when the target version already exists in the npm registry and still writes the release record', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      publishRegistry: 'https://npm.example.com',
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: ['commands'],
      errors: [],
    });
    vi.mocked(packLocalSkillDirectory).mockResolvedValueOnce({
      bundlePath: join(testDir, 'packed.tgz'),
      manifestDigest: 'sha256:packed',
      cleanup: vi.fn(),
    });
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            name: '@acme.skill/my-skill',
            versions: {
              '1.2.0': {
                name: '@acme.skill/my-skill',
                version: '1.2.0',
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

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(spawnSync).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://npm.example.com/%40acme.skill%2Fmy-skill',
      expect.objectContaining({ method: 'GET' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://catalog.example.com/api/v1/skills/%40acme.skill%2Fmy-skill/versions/1.2.0/release-record',
      expect.objectContaining({ method: 'PUT' })
    );
    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      bundlePath: join(testDir, 'packed.tgz'),
    });
  });

  it('recovers from an already-published npm version by reconciling the release record', async () => {
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 1,
      stdout: '',
      stderr: 'npm error You cannot publish over the previously published versions: 1.2.0.',
    } as ReturnType<typeof spawnSync>);
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      publishRegistry: 'https://npm.example.com',
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: ['commands'],
      errors: [],
    });
    vi.mocked(packLocalSkillDirectory).mockResolvedValueOnce({
      bundlePath: join(testDir, 'packed.tgz'),
      manifestDigest: 'sha256:packed',
      cleanup: vi.fn(),
    });
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            name: '@acme.skill/my-skill',
            versions: {},
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            name: '@acme.skill/my-skill',
            versions: {
              '1.2.0': {
                name: '@acme.skill/my-skill',
                version: '1.2.0',
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

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(spawnSync).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://npm.example.com/%40acme.skill%2Fmy-skill',
      expect.objectContaining({ method: 'GET' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://npm.example.com/%40acme.skill%2Fmy-skill',
      expect.objectContaining({ method: 'GET' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      'https://catalog.example.com/api/v1/skills/%40acme.skill%2Fmy-skill/versions/1.2.0/release-record',
      expect.objectContaining({ method: 'PUT' })
    );
    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      bundlePath: join(testDir, 'packed.tgz'),
    });
  });

  it('prefers the externally injected npm userconfig and registry when publishing', async () => {
    writeFileSync(join(testDir, '.npmrc'), '//npm.example.com/:_authToken=local-token\n');
    process.env.NPM_CONFIG_USERCONFIG = '/tmp/external.npmrc';
    process.env.npm_config_registry = 'http://127.0.0.1:8081/repository/skills-npm/';
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      publishRegistry: undefined,
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: ['commands'],
      errors: [],
    });
    vi.mocked(packLocalSkillDirectory).mockResolvedValueOnce({
      bundlePath: join(testDir, 'packed.tgz'),
      manifestDigest: 'sha256:packed',
      cleanup: vi.fn(),
    });
    fetchMock
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

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result.ok).toBe(true);
    const publishCall = vi.mocked(spawnSync).mock.calls[0];
    expect(publishCall?.[1]).toEqual([
      'publish',
      join(testDir, 'packed.tgz'),
      '--registry',
      'http://127.0.0.1:8081/repository/skills-npm/',
    ]);
    expect(publishCall?.[2]).toMatchObject({
      env: expect.objectContaining({
        NPM_CONFIG_USERCONFIG: '/tmp/external.npmrc',
      }),
    });
  });

  it('normalizes a trailing slash in publishRegistry before writing the release record', async () => {
    vi.mocked(spawnSync).mockReturnValueOnce({
      status: 0,
      stdout: 'published',
      stderr: '',
    } as ReturnType<typeof spawnSync>);
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      publishRegistry: 'https://npm.example.com/',
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: ['commands'],
      errors: [],
    });
    vi.mocked(packLocalSkillDirectory).mockResolvedValueOnce({
      bundlePath: join(testDir, 'packed.tgz'),
      manifestDigest: 'sha256:packed',
      cleanup: vi.fn(),
    });
    fetchMock
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

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result.ok).toBe(true);
    const releaseRecordPayload = JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body));
    expect(releaseRecordPayload).toEqual({
      publishRegistry: 'https://npm.example.com',
      manifestDigest: 'sha256:packed',
      components: ['commands'],
    });
    const publishCall = vi.mocked(spawnSync).mock.calls[0];
    expect(publishCall?.[1]).toEqual([
      'publish',
      join(testDir, 'packed.tgz'),
      '--registry',
      'https://npm.example.com/',
    ]);
  });

  it('returns local packer errors without publishing or writing release records', async () => {
    vi.mocked(runPackagingSession).mockResolvedValueOnce({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      publishRegistry: 'https://npm.example.com',
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      displayName: 'my-skill',
      description: 'A valid skill',
      components: ['commands'],
      errors: [],
    });
    vi.mocked(packLocalSkillDirectory).mockRejectedValueOnce(new Error('unsafe_path_entry'));

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      errors: ['unsafe_path_entry'],
    });
    expect(spawnSync).not.toHaveBeenCalled();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
