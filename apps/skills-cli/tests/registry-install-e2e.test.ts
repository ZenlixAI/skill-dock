import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { runAdd } from '../src/add.ts';

const { startMockCatalogServer } = await import(
  new URL('../../../test-support/mock-catalog-server.mjs', import.meta.url).href
);

describe('registry install e2e', () => {
  let server: Awaited<ReturnType<typeof startMockCatalogServer>>;
  let originalCwd: string;

  beforeAll(async () => {
    server = await startMockCatalogServer({ port: 0 });
  });

  beforeEach(() => {
    originalCwd = process.cwd();
    server.requests.length = 0;
    vi.stubEnv('SKILLS_FORCE_HUMAN_CLI', '1');
  });

  afterEach(() => {
    process.chdir(originalCwd);
    vi.unstubAllEnvs();
  });

  afterAll(async () => {
    await server.close();
  });

  it('downloads proxy bundles without npm auth', async () => {
    const cwd = mkdirTempDir('skills-cli-registry-e2e-');
    const outputDir = join(cwd, 'output');
    mkdirSync(outputDir, { recursive: true });
    process.chdir(cwd);

    try {
      await runAdd(['@platform.skill/agent-notes'], {
        version: '1.0.1',
        registry: server.baseUrl,
        outputDir,
        json: true,
        yes: true,
      });

      expect(JSON.parse(readFileSync(join(outputDir, '.skill-manifest.json'), 'utf-8'))).toMatchObject({
        packageName: '@platform.skill/agent-notes',
        resolvedVersion: '1.0.1',
        installName: 'agent-notes',
      });
      expect(server.requests).toEqual([
        expect.objectContaining({
          packageName: '@platform.skill/agent-notes',
          version: '1.0.1',
          authorization: null,
          path: '/repository/skills-npm/@platform.skill/agent-notes/-/agent-notes-1.0.1.tgz',
        }),
      ]);
      expect(existsSync(join(outputDir, '.skill-manifest.json'))).toBe(true);
    } finally {
      rmSync(cwd, { recursive: true, force: true });
    }
  });

  it('downloads direct bundles with npm auth from .npmrc', async () => {
    const cwd = mkdirTempDir('skills-cli-registry-e2e-');
    const outputDir = join(cwd, 'output');
    mkdirSync(outputDir, { recursive: true });
    writeFileSync(join(cwd, '.npmrc'), `//127.0.0.1:${server.port}/direct/:_authToken=fixture-token\n`);
    process.chdir(cwd);

    try {
      await runAdd(['@platform.skill/agent-notes'], {
        version: '1.0.0',
        registry: server.baseUrl,
        outputDir,
        json: true,
        yes: true,
      });

      expect(JSON.parse(readFileSync(join(outputDir, '.skill-manifest.json'), 'utf-8'))).toMatchObject({
        packageName: '@platform.skill/agent-notes',
        resolvedVersion: '1.0.0',
        installName: 'agent-notes',
      });
      expect(server.requests).toEqual([
        expect.objectContaining({
          packageName: '@platform.skill/agent-notes',
          version: '1.0.0',
          authorization: 'Bearer fixture-token',
          path: '/direct/@platform.skill/agent-notes/-/agent-notes-1.0.0.tgz',
        }),
      ]);
    } finally {
      rmSync(cwd, { recursive: true, force: true });
    }
  });
});

function mkdirTempDir(prefix: string): string {
  const dir = join(tmpdir(), `${prefix}${Date.now()}-${Math.random().toString(16).slice(2)}`);
  mkdirSync(dir, { recursive: true });
  return dir;
}
