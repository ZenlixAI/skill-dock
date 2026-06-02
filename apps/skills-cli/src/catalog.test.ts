import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getRegistryBaseUrl,
  packSkillPackaging,
  prepareSkillPackaging,
  putReleaseRecord,
  resolveInstallTarget,
  searchCatalogSkillsPage,
  searchCatalogSkills,
  validateSkillPackaging,
} from './catalog.ts';

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

describe('catalog client', () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    fetchMock = vi.fn();
    globalThis.fetch = fetchMock as unknown as typeof globalThis.fetch;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.unstubAllEnvs();
  });

  it('builds the registry base URL from env when unset explicitly', () => {
    vi.stubEnv('SKILLS_REGISTRY_URL', 'https://catalog.example.com/');
    expect(getRegistryBaseUrl()).toBe('https://catalog.example.com');
  });

  it('does not fall back to the removed SKILLS_API_URL alias', () => {
    vi.stubEnv('SKILLS_API_URL', 'https://legacy.example.com/');
    expect(getRegistryBaseUrl()).toBe('https://skills.sh');
  });

  it('searches the catalog with q against /api/v1/skills', async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        items: [
          {
            packageName: '@platform.skill/pdf-skill',
            displayName: 'PDF Skill',
            description: 'PDF processing workflow',
            latestVersion: '1.2.0',
          },
        ],
      })
    );

    const result = await searchCatalogSkills('pdf', {
      registry: 'https://catalog.example.com/root/',
    });

    expect(fetchMock).toHaveBeenCalledWith(
      'https://catalog.example.com/root/api/v1/skills?q=pdf&pageSize=10&sort=relevance',
      expect.any(Object)
    );
    expect(result).toEqual([
      {
        packageName: '@platform.skill/pdf-skill',
        displayName: 'PDF Skill',
        description: 'PDF processing workflow',
        latestVersion: '1.2.0',
      },
    ]);
  });

  it('supports paged catalog search and preserves pageInfo cursor tokens', async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        items: [
          {
            packageName: '@platform.skill/pdf-skill',
            displayName: 'PDF Skill',
            description: 'PDF processing workflow',
            latestVersion: '1.2.0',
          },
        ],
        pageInfo: {
          nextCursor: 'cursor_1',
          hasNextPage: true,
        },
      })
    );

    const result = await searchCatalogSkillsPage('pdf', {
      registry: 'https://catalog.example.com/root/',
      pageSize: 1,
      cursor: 'cursor_0',
    });

    expect(fetchMock).toHaveBeenCalledWith(
      'https://catalog.example.com/root/api/v1/skills?q=pdf&pageSize=1&cursor=cursor_0&sort=relevance',
      expect.any(Object)
    );
    expect(result).toEqual({
      items: [
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          description: 'PDF processing workflow',
          latestVersion: '1.2.0',
        },
      ],
      pageInfo: {
        nextCursor: 'cursor_1',
        hasNextPage: true,
      },
    });
  });

  it('resolves install targets with URL-encoded package names and exact version', async () => {
    vi.stubEnv('SKILLS_REGISTRY_TOKEN', 'publisher-token');
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-test',
        versionStatus: 'published',
      })
    );

    const result = await resolveInstallTarget('@platform.skill/pdf-skill', {
      version: '1.2.0',
      registry: 'https://catalog.example.com',
    });

    expect(fetchMock).toHaveBeenCalledWith(
      'https://catalog.example.com/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target?version=1.2.0',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer publisher-token',
        }),
      })
    );
    expect(result.packageName).toBe('@platform.skill/pdf-skill');
    expect(result.version).toBe('1.2.0');
  });

  it('sends an X-Role header when configured for local dev registry auth', async () => {
    vi.stubEnv('SKILLS_REGISTRY_ROLE', 'publisher');

    fetchMock.mockResolvedValueOnce(jsonResponse({ ok: true }));

    await putReleaseRecord('@platform.skill/pdf-skill', '1.2.0', {
      registry: 'https://catalog.example.com',
    });

    expect(fetchMock).toHaveBeenCalledWith(
      'https://catalog.example.com/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record',
      expect.objectContaining({
        headers: expect.objectContaining({
          'X-Role': 'publisher',
        }),
      })
    );
  });

  it('calls prepare/validate/pack packaging endpoints', async () => {
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          workspacePath: '/tmp/workspace',
          packageName: '@platform.skill/pdf-skill',
          version: '1.2.0',
          warnings: [],
          errors: [],
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          ok: true,
          workspacePath: '/tmp/workspace',
          packageName: '@platform.skill/pdf-skill',
          version: '1.2.0',
          manifestDigest: 'sha256:manifest',
          components: ['commands'],
          warnings: [],
          errors: [],
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          workspacePath: '/tmp/workspace',
          bundlePath: '/tmp/workspace/pdf-skill-1.2.0.tgz',
          packageName: '@platform.skill/pdf-skill',
          version: '1.2.0',
          manifestDigest: 'sha256:manifest',
          components: ['commands'],
          warnings: [],
          errors: [],
        })
      );

    await prepareSkillPackaging({
      registry: 'https://catalog.example.com',
      source: { type: 'local', location: '/tmp/skill' },
      target: { packageName: '@platform.skill/pdf-skill', version: '1.2.0' },
    });
    await validateSkillPackaging({
      registry: 'https://catalog.example.com',
      workspacePath: '/tmp/workspace',
      target: { packageName: '@platform.skill/pdf-skill', version: '1.2.0' },
    });
    await packSkillPackaging({
      registry: 'https://catalog.example.com',
      workspacePath: '/tmp/workspace',
      target: { packageName: '@platform.skill/pdf-skill', version: '1.2.0' },
      includeBundle: true,
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://catalog.example.com/api/v1/skill-packaging/prepare',
      expect.objectContaining({ method: 'POST' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://catalog.example.com/api/v1/skill-packaging/validate',
      expect.objectContaining({ method: 'POST' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      'https://catalog.example.com/api/v1/skill-packaging/pack',
      expect.objectContaining({ method: 'POST' })
    );
  });

  it('surfaces stable catalog errors', async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(
        {
          error: {
            code: 'skill_not_found',
            message: 'Skill not found.',
            details: { packageName: '@platform.skill/missing-skill' },
            requestId: 'req_123',
          },
        },
        404
      )
    );

    await expect(
      resolveInstallTarget('@platform.skill/missing-skill', {
        registry: 'https://catalog.example.com',
      })
    ).rejects.toMatchObject({
      code: 'skill_not_found',
      requestId: 'req_123',
    });
  });
});
