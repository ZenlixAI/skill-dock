import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync, readFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { execFileSync } from 'child_process';
import { createHash } from 'crypto';
import { downloadRegistrySkillBundle } from './registry-bundle.ts';

describe('downloadRegistrySkillBundle', () => {
  let fixtureDir: string;
  let originalFetch: typeof globalThis.fetch;

  beforeEach(() => {
    fixtureDir = mkdtempSync(join(tmpdir(), 'skills-registry-bundle-test-'));
    originalFetch = globalThis.fetch;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    rmSync(fixtureDir, { recursive: true, force: true });
  });

  function sha512Integrity(content: Buffer): string {
    return `sha512-${createHash('sha512').update(content).digest('base64')}`;
  }

  function skillManifestDigest(content: string): string {
    return `sha256:${createHash('sha256').update(content).digest('hex')}`;
  }

  function createBundleFixture(options?: { packageName?: string; version?: string }) {
    const packageDir = join(fixtureDir, 'package');
    mkdirSync(packageDir, { recursive: true });

    const skillContent = `---
name: pdf-skill
description: PDF processing workflow
---

# PDF Skill
`;
    writeFileSync(join(packageDir, 'SKILL.md'), skillContent);
    writeFileSync(
      join(packageDir, 'package.json'),
      JSON.stringify(
        {
          name: options?.packageName || '@platform.skill/pdf-skill',
          version: options?.version || '1.2.0',
        },
        null,
        2
      )
    );

    const tarballPath = join(fixtureDir, 'bundle.tgz');
    execFileSync('tar', ['-czf', tarballPath, '-C', fixtureDir, 'package']);
    const tarball = readFileSync(tarballPath);

    return {
      tarball,
      skillContent,
    };
  }

  it('downloads, extracts, and parses a registry skill bundle', async () => {
    const fixture = createBundleFixture();

    globalThis.fetch = vi.fn(async () => {
      return new Response(fixture.tarball, {
        status: 200,
        headers: { 'content-type': 'application/octet-stream' },
      });
    }) as unknown as typeof globalThis.fetch;

    const bundle = await downloadRegistrySkillBundle({
      packageName: '@platform.skill/pdf-skill',
      version: '1.2.0',
      npmRegistry: 'https://npm.example.com',
      tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
      integrity: sha512Integrity(fixture.tarball),
      skillManifestDigest: skillManifestDigest(fixture.skillContent),
      versionStatus: 'published',
    });

    try {
      expect(bundle.skill.name).toBe('@platform.skill/pdf-skill');
      expect(bundle.skill.description).toBe('PDF processing workflow');
      expect(bundle.extractedPath).toContain('/package');
    } finally {
      await bundle.cleanup();
    }
  });

  it('rejects a bundle whose tarball integrity does not match the install target', async () => {
    const fixture = createBundleFixture();

    globalThis.fetch = vi.fn(async () => {
      return new Response(fixture.tarball, {
        status: 200,
        headers: { 'content-type': 'application/octet-stream' },
      });
    }) as unknown as typeof globalThis.fetch;

    await expect(
      downloadRegistrySkillBundle({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
        integrity: 'sha512-invalid',
        skillManifestDigest: skillManifestDigest(fixture.skillContent),
        versionStatus: 'published',
      })
    ).rejects.toThrow('Resolved bundle failed integrity verification');
  });

  it('rejects a bundle whose SKILL.md digest does not match the install target', async () => {
    const fixture = createBundleFixture();

    globalThis.fetch = vi.fn(async () => {
      return new Response(fixture.tarball, {
        status: 200,
        headers: { 'content-type': 'application/octet-stream' },
      });
    }) as unknown as typeof globalThis.fetch;

    await expect(
      downloadRegistrySkillBundle({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
        integrity: sha512Integrity(fixture.tarball),
        skillManifestDigest: 'sha256:invalid',
        versionStatus: 'published',
      })
    ).rejects.toThrow('Resolved bundle failed manifest digest verification');
  });

  it('rejects a bundle whose package.json metadata does not match the install target', async () => {
    const fixture = createBundleFixture({ version: '9.9.9' });

    globalThis.fetch = vi.fn(async () => {
      return new Response(fixture.tarball, {
        status: 200,
        headers: { 'content-type': 'application/octet-stream' },
      });
    }) as unknown as typeof globalThis.fetch;

    await expect(
      downloadRegistrySkillBundle({
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        npmRegistry: 'https://npm.example.com',
        tarballUrl: 'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
        integrity: sha512Integrity(fixture.tarball),
        skillManifestDigest: skillManifestDigest(fixture.skillContent),
        versionStatus: 'published',
      })
    ).rejects.toThrow('Resolved bundle package metadata does not match the install target');
  });
});
