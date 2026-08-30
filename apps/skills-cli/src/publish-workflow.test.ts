import { existsSync, mkdirSync, rmSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

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

  it('publishes directly through Catalog without npm publish or npm auth', async () => {
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
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          packageName: '@acme.skill/my-skill',
          version: '1.2.0',
          manifestDigest: 'sha256:packed',
          components: ['commands'],
          publishedToRegistry: true,
          created: true,
          idempotentReplay: false,
          versionStatus: 'published',
          installPolicy: 'resolvable',
          warnings: [],
          errors: [],
        }),
        {
          status: 201,
          headers: { 'content-type': 'application/json' },
        }
      )
    );

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(packLocalSkillDirectory).toHaveBeenCalledWith({ directory: testDir });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      'https://catalog.example.com/api/v1/skill-packaging/publish',
      expect.objectContaining({ method: 'POST' })
    );
    const publishPayload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(publishPayload).toMatchObject({
      target: {
        packageName: '@acme.skill/my-skill',
        version: '1.2.0',
      },
      metadata: {
        displayName: 'my-skill',
        description: 'A valid skill',
        sourceType: 'native',
      },
      options: {},
      review: {
        manifestDigest: 'sha256:packed',
        components: ['commands'],
      },
    });
    expect(publishPayload.source.type).toBe('upload_archive');
    expect(typeof publishPayload.source.archiveBase64).toBe('string');
    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/my-skill',
      version: '1.2.0',
      bundlePath: join(testDir, 'packed.tgz'),
    });
  });

  it('requires a catalog registry when no override is provided', async () => {
    vi.stubEnv('SKILLS_REGISTRY_URL', '');
    await expect(runPublishWorkflow({ path: testDir })).rejects.toThrow(/registry URL is required/);
    expect(runPackagingSession).not.toHaveBeenCalled();
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
    expect(packLocalSkillDirectory).not.toHaveBeenCalled();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('returns publish failures from Catalog without a release record side effect', async () => {
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
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          error: {
            code: 'internal_error',
            message: 'Catalog publish failed.',
          },
        }),
        {
          status: 500,
          headers: { 'content-type': 'application/json' },
        }
      )
    );

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      errors: ['Catalog publish failed.'],
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('returns local packer errors without publishing or writing release records', async () => {
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
    vi.mocked(packLocalSkillDirectory).mockRejectedValueOnce(new Error('unsafe_path_entry'));

    const result = await runPublishWorkflow({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      errors: ['unsafe_path_entry'],
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
