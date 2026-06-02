import { existsSync, mkdirSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { runPackagingSession } from './packaging-session.ts';

describe('packaging session', () => {
  let testDir: string;
  let originalFetch: typeof globalThis.fetch;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    testDir = join(tmpdir(), `skills-packaging-session-test-${Date.now()}`);
    mkdirSync(testDir, { recursive: true });

    originalFetch = globalThis.fetch;
    fetchMock = vi.fn();
    globalThis.fetch = fetchMock as unknown as typeof globalThis.fetch;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('runs prepare and validate with a shared idempotency key', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/my-skill', version: '1.2.0' }, null, 2)
    );
    mkdirSync(join(testDir, 'commands'));

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
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
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            manifestDigest: 'sha256:test',
            components: ['commands'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      );

    const result = await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toMatchObject({
      ok: true,
      sessionKind: 'registry',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      workspacePath: '/tmp/workspace',
      manifestDigest: 'sha256:test',
      components: ['commands'],
    });
    expect('idempotencyKey' in result).toBe(true);
    if (result.ok && result.sessionKind === 'registry') {
      expect(typeof result.idempotencyKey).toBe('string');
      expect(result.idempotencyKey.length).toBeGreaterThan(0);
    }
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

    const preparePayload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    const validatePayload = JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body));
    expect(preparePayload.idempotencyKey).toBeTruthy();
    expect(validatePayload.idempotencyKey).toBe(preparePayload.idempotencyKey);
    expect(preparePayload.metadata).toEqual({
      displayName: 'my-skill',
      description: 'A valid skill',
      sourceType: 'native',
    });
  });

  it('returns local inspection errors without calling packaging endpoints', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: broken-skill
description: Missing version
---
`
    );
    writeFileSync(join(testDir, 'package.json'), JSON.stringify({ name: 'broken-skill' }, null, 2));

    const result = await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'broken-skill',
      description: 'Missing version',
      publishRegistry: undefined,
      components: [],
      errors: [
        'package.json name must use the reserved @platform.skill/ scope',
        'package.json version is required',
      ],
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('returns a full success surface for local-only inspection', async () => {
    mkdirSync(join(testDir, 'commands'));
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: local-skill
description: Local-only inspect
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/local-skill', version: '1.2.0' }, null, 2)
    );

    const result = await runPackagingSession({
      path: testDir,
    });

    expect(result).toEqual({
      ok: true,
      sessionKind: 'local',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'local-skill',
      description: 'Local-only inspect',
      packageName: '@platform.skill/local-skill',
      version: '1.2.0',
      publishRegistry: undefined,
      components: ['commands'],
      errors: [],
    });
    expect('workspacePath' in result).toBe(false);
    expect('idempotencyKey' in result).toBe(false);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('returns prepare errors without calling validate', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/my-skill', version: '1.2.0' }, null, 2)
    );

    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          workspacePath: '/tmp/workspace',
          packageName: '@platform.skill/my-skill',
          version: '1.2.0',
          warnings: [],
          errors: ['nested_skill_md'],
        }),
        { status: 200, headers: { 'content-type': 'application/json' } }
      )
    );

    const result = await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'my-skill',
      description: 'A valid skill',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      publishRegistry: undefined,
      components: [],
      errors: ['nested_skill_md'],
      workspacePath: '/tmp/workspace',
      idempotencyKey: expect.any(String),
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('preserves an explicit package sourceType when preparing registry packaging', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify(
        {
          name: '@platform.skill/my-skill',
          version: '1.2.0',
          skill: {
            sourceType: 'platform_created',
          },
        },
        null,
        2
      )
    );
    mkdirSync(join(testDir, 'commands'));

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
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
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            manifestDigest: 'sha256:test',
            components: ['commands'],
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      );

    await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    const preparePayload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(preparePayload.metadata).toEqual({
      displayName: 'my-skill',
      description: 'A valid skill',
      sourceType: 'platform_created',
    });
  });

  it('returns validate failures with the session workspace and idempotency key', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/my-skill', version: '1.2.0' }, null, 2)
    );

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ok: false,
            workspacePath: '/tmp/workspace',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            components: [],
            warnings: [],
            errors: ['manifest_digest_mismatch'],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      );

    const result = await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'my-skill',
      description: 'A valid skill',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      publishRegistry: undefined,
      components: [],
      errors: ['manifest_digest_mismatch'],
      workspacePath: '/tmp/workspace',
      idempotencyKey: expect.any(String),
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('normalizes thrown catalog errors into a failure result', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/my-skill', version: '1.2.0' }, null, 2)
    );

    fetchMock.mockRejectedValueOnce(new Error('network down'));

    const result = await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'my-skill',
      description: 'A valid skill',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      publishRegistry: undefined,
      components: [],
      errors: ['network down'],
      idempotencyKey: expect.any(String),
    });
    expect('workspacePath' in result).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('normalizes malformed prepare success payloads into a failure result', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/my-skill', version: '1.2.0' }, null, 2)
    );

    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          workspacePath: '/tmp/workspace',
          packageName: '@platform.skill/my-skill',
          version: '1.2.0',
          warnings: [],
        }),
        { status: 200, headers: { 'content-type': 'application/json' } }
      )
    );

    const result = await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'my-skill',
      description: 'A valid skill',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      publishRegistry: undefined,
      components: [],
      errors: ['catalog prepare returned malformed success payload'],
      idempotencyKey: expect.any(String),
    });
    expect('workspacePath' in result).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('normalizes malformed validate success payloads into a failure result', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/my-skill', version: '1.2.0' }, null, 2)
    );

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
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
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            warnings: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      );

    const result = await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'my-skill',
      description: 'A valid skill',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      publishRegistry: undefined,
      components: [],
      errors: ['catalog validate returned malformed success payload'],
      workspacePath: '/tmp/workspace',
      idempotencyKey: expect.any(String),
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('normalizes thrown validate errors into a failure result', async () => {
    writeFileSync(
      join(testDir, 'SKILL.md'),
      `---
name: my-skill
description: A valid skill
---
`
    );
    writeFileSync(
      join(testDir, 'package.json'),
      JSON.stringify({ name: '@platform.skill/my-skill', version: '1.2.0' }, null, 2)
    );

    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            workspacePath: '/tmp/workspace',
            packageName: '@platform.skill/my-skill',
            version: '1.2.0',
            warnings: [],
            errors: [],
          }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
      )
      .mockRejectedValueOnce(new Error('validation down'));

    const result = await runPackagingSession({
      path: testDir,
      registry: 'https://catalog.example.com',
    });

    expect(result).toEqual({
      ok: false,
      sessionKind: 'registry',
      directory: testDir,
      skillMdPath: join(testDir, 'SKILL.md'),
      packageJsonPath: join(testDir, 'package.json'),
      displayName: 'my-skill',
      description: 'A valid skill',
      packageName: '@platform.skill/my-skill',
      version: '1.2.0',
      publishRegistry: undefined,
      components: [],
      errors: ['validation down'],
      workspacePath: '/tmp/workspace',
      idempotencyKey: expect.any(String),
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
