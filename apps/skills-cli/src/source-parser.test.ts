import { mkdirSync, rmSync, mkdtempSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, describe, expect, it } from 'vitest';
import { parseSourceInput } from './source-parser.ts';

describe('source parser', () => {
  const tempDirs: string[] = [];

  afterEach(() => {
    for (const dir of tempDirs) {
      rmSync(dir, { recursive: true, force: true });
    }
    tempDirs.length = 0;
  });

  it('parses GitHub shorthand with an inline skill filter', () => {
    expect(parseSourceInput('owner/repo@pdf')).toEqual({
      type: 'github',
      url: 'https://github.com/owner/repo.git',
      skillFilter: 'pdf',
    });
  });

  it('parses GitHub URLs and preserves ref/path selectors', () => {
    expect(parseSourceInput('https://github.com/owner/repo?path=skills/pdf&ref=main')).toEqual({
      type: 'github',
      url: 'https://github.com/owner/repo.git',
      subpath: 'skills/pdf',
      ref: 'main',
    });
  });

  it('parses generic git URLs', () => {
    expect(parseSourceInput('https://example.com/org/repo.git?ref=v1')).toEqual({
      type: 'git',
      url: 'https://example.com/org/repo.git?ref=v1',
      ref: 'v1',
    });
  });

  it('parses local paths', () => {
    const directory = mkdtempSync(join(tmpdir(), 'skills-source-parser-'));
    mkdirSync(directory, { recursive: true });
    tempDirs.push(directory);

    expect(parseSourceInput(directory)).toEqual({
      type: 'local',
      url: directory,
      localPath: directory,
    });
  });

  it('resolves local paths from INIT_CWD when launched through npx', () => {
    const cwd = mkdtempSync(join(tmpdir(), 'skills-source-parser-cwd-'));
    const initCwd = mkdtempSync(join(tmpdir(), 'skills-source-parser-init-'));
    const localDirName = 'my-aaa';
    const localDir = join(initCwd, localDirName);
    mkdirSync(cwd, { recursive: true });
    mkdirSync(localDir, { recursive: true });
    tempDirs.push(cwd, initCwd);

    const previousCwd = process.cwd();
    const previousInitCwd = process.env.INIT_CWD;
    process.chdir(cwd);
    process.env.INIT_CWD = initCwd;
    try {
      expect(parseSourceInput(localDirName)).toEqual({
        type: 'local',
        url: localDir,
        localPath: localDir,
      });
    } finally {
      process.chdir(previousCwd);
      process.env.INIT_CWD = previousInitCwd;
    }
  });
});
