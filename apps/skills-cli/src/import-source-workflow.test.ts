import { existsSync, mkdirSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./git.ts', () => ({
  cloneRepo: vi.fn(),
  cleanupTempDir: vi.fn(() => Promise.resolve()),
}));

import { runImportGitWorkflow } from './import-git-workflow.ts';
import { cloneRepo, cleanupTempDir } from './git.ts';

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

describe('import-git workflow', () => {
  let repoDir: string;
  let originalFetch: typeof globalThis.fetch;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    repoDir = join(tmpdir(), `skills-import-source-workflow-test-${Date.now()}`);
    mkdirSync(repoDir, { recursive: true });

    originalFetch = globalThis.fetch;
    fetchMock = vi.fn();
    globalThis.fetch = fetchMock as unknown as typeof globalThis.fetch;

    vi.mocked(cloneRepo).mockReset();
    vi.mocked(cleanupTempDir).mockClear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
    if (existsSync(repoDir)) {
      rmSync(repoDir, { recursive: true, force: true });
    }
  });

  it('clones an external source, discovers one skill, and publishes it through the local import flow', async () => {
    const skillDir = join(repoDir, 'skills', 'legacy-skill');
    mkdirSync(join(skillDir, 'docs'), { recursive: true });
    writeFileSync(
      join(skillDir, 'SKILL.md'),
      `---\nname: legacy-skill\ndescription: Imported skill\n---\n`
    );

    vi.mocked(cloneRepo).mockResolvedValueOnce(repoDir);
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

    const result = await runImportGitWorkflow({
      source: 'owner/repo',
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
    });

    expect(cloneRepo).toHaveBeenCalledWith('https://github.com/owner/repo.git', undefined);
    expect(fetchMock).toHaveBeenCalledWith(
      'https://catalog.example.com/api/v1/skill-packaging/publish',
      expect.objectContaining({ method: 'POST' })
    );
    expect(result).toEqual({
      ok: true,
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
      manifestDigest: 'sha256:test',
      components: ['docs'],
      publishedToRegistry: true,
      created: true,
      idempotentReplay: false,
      versionStatus: 'published',
      installPolicy: 'resolvable',
    });
    expect(cleanupTempDir).toHaveBeenCalledWith(repoDir);
  });

  it('requires --skill when multiple skills are discovered', async () => {
    const firstSkill = join(repoDir, 'skills', 'first');
    const secondSkill = join(repoDir, 'skills', 'second');
    mkdirSync(firstSkill, { recursive: true });
    mkdirSync(secondSkill, { recursive: true });
    writeFileSync(join(firstSkill, 'SKILL.md'), `---\nname: first\ndescription: First\n---\n`);
    writeFileSync(join(secondSkill, 'SKILL.md'), `---\nname: second\ndescription: Second\n---\n`);

    vi.mocked(cloneRepo).mockResolvedValueOnce(repoDir);

    const result = await runImportGitWorkflow({
      source: 'owner/repo',
      registry: 'https://catalog.example.com',
      packageName: '@acme.skill/legacy-skill',
      version: '0.1.0',
    });

    expect(result).toEqual({
      ok: false,
      errors: ['Multiple skills found. Pass --skill <name> to choose one: first, second'],
    });
  });
});
