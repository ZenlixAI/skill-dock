import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { readSkillLock, writeSkillLock } from '../src/skill-lock.ts';

describe('skill-lock', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it('preserves older lock entries instead of wiping them on version upgrades', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'skill-lock-test-'));
    process.env.XDG_STATE_HOME = dir;

    try {
      await mkdir(join(dir, 'skills'), { recursive: true });
      await writeFile(
        join(dir, 'skills', '.skill-lock.json'),
        JSON.stringify(
          {
            version: 2,
            skills: {
              '@platform.skill/pdf-skill': {
                source: '@platform.skill/pdf-skill',
                sourceType: 'registry',
                sourceUrl:
                  'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.0.0.tgz',
                packageName: '@platform.skill/pdf-skill',
                version: '1.0.0',
                installedAt: '2026-01-01T00:00:00.000Z',
                updatedAt: '2026-01-01T00:00:00.000Z',
              },
            },
            dismissed: {
              findSkillsPrompt: true,
            },
            lastSelectedAgents: ['claude-code'],
          },
          null,
          2
        ),
        'utf-8'
      );

      const lock = await readSkillLock();
      expect(lock.version).toBe(3);
      expect(lock.skills['@platform.skill/pdf-skill']).toMatchObject({
        packageName: '@platform.skill/pdf-skill',
        version: '1.0.0',
        sourceType: 'registry',
        legacy: {
          source: '@platform.skill/pdf-skill',
          sourceUrl:
            'https://npm.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.0.0.tgz',
        },
      });
      expect(lock.dismissed?.findSkillsPrompt).toBe(true);
      expect(lock.lastSelectedAgents).toEqual(['claude-code']);
    } finally {
      await rm(dir, { recursive: true, force: true });
    }
  });

  it('accepts current-version registry entries without compatibility source fields', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'skill-lock-test-'));
    process.env.XDG_STATE_HOME = dir;

    try {
      await mkdir(join(dir, 'skills'), { recursive: true });
      await writeFile(
        join(dir, 'skills', '.skill-lock.json'),
        JSON.stringify(
          {
            version: 3,
            skills: {
              '@platform.skill/pdf-skill': {
                sourceType: 'registry',
                packageName: '@platform.skill/pdf-skill',
                version: '1.2.0',
                registry: 'https://catalog.example.com',
                integrity: 'sha512-test',
                skillManifestDigest: 'sha256-manifest',
                skillFolderHash: 'folder-hash',
                installedAt: '2026-01-01T00:00:00.000Z',
                updatedAt: '2026-01-01T00:00:00.000Z',
              },
            },
          },
          null,
          2
        ),
        'utf-8'
      );

      const lock = await readSkillLock();
      expect(lock.skills['@platform.skill/pdf-skill']).toEqual({
        sourceType: 'registry',
        packageName: '@platform.skill/pdf-skill',
        version: '1.2.0',
        registry: 'https://catalog.example.com',
        integrity: 'sha512-test',
        skillManifestDigest: 'sha256-manifest',
        skillFolderHash: 'folder-hash',
        installedAt: '2026-01-01T00:00:00.000Z',
        updatedAt: '2026-01-01T00:00:00.000Z',
      });
    } finally {
      await rm(dir, { recursive: true, force: true });
    }
  });

  it('normalizes current-version historical fields into legacy metadata', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'skill-lock-test-'));
    process.env.XDG_STATE_HOME = dir;

    try {
      await mkdir(join(dir, 'skills'), { recursive: true });
      await writeFile(
        join(dir, 'skills', '.skill-lock.json'),
        JSON.stringify(
          {
            version: 3,
            skills: {
              'legacy-skill': {
                source: 'owner/repo',
                sourceType: 'github',
                sourceUrl: 'https://github.com/owner/repo.git',
                ref: 'main',
                skillPath: 'skills/legacy-skill/SKILL.md',
                skillFolderHash: 'folder-hash',
                installedAt: '2026-01-01T00:00:00.000Z',
                updatedAt: '2026-01-01T00:00:00.000Z',
              },
            },
          },
          null,
          2
        ),
        'utf-8'
      );

      const lock = await readSkillLock();
      expect(lock.skills['legacy-skill']).toEqual({
        sourceType: 'github',
        skillFolderHash: 'folder-hash',
        installedAt: '2026-01-01T00:00:00.000Z',
        updatedAt: '2026-01-01T00:00:00.000Z',
        legacy: {
          source: 'owner/repo',
          sourceUrl: 'https://github.com/owner/repo.git',
          ref: 'main',
          skillPath: 'skills/legacy-skill/SKILL.md',
        },
      });
    } finally {
      await rm(dir, { recursive: true, force: true });
    }
  });

  it('serializes legacy metadata using historical top-level keys', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'skill-lock-test-'));
    process.env.XDG_STATE_HOME = dir;

    try {
      await writeSkillLock({
        version: 3,
        skills: {
          'legacy-skill': {
            sourceType: 'github',
            skillFolderHash: 'folder-hash',
            installedAt: '2026-01-01T00:00:00.000Z',
            updatedAt: '2026-01-01T00:00:00.000Z',
            legacy: {
              source: 'owner/repo',
              sourceUrl: 'https://github.com/owner/repo.git',
              ref: 'main',
              skillPath: 'skills/legacy-skill/SKILL.md',
            },
          },
        },
      });

      const raw = JSON.parse(await readFile(join(dir, 'skills', '.skill-lock.json'), 'utf-8'));
      expect(raw.skills['legacy-skill']).toEqual({
        source: 'owner/repo',
        sourceUrl: 'https://github.com/owner/repo.git',
        ref: 'main',
        skillPath: 'skills/legacy-skill/SKILL.md',
        sourceType: 'github',
        skillFolderHash: 'folder-hash',
        installedAt: '2026-01-01T00:00:00.000Z',
        updatedAt: '2026-01-01T00:00:00.000Z',
      });
    } finally {
      await rm(dir, { recursive: true, force: true });
    }
  });
});
