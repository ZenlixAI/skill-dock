import { mkdirSync, rmSync } from 'fs';
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
    const directory = join(tmpdir(), `skills-source-parser-${Date.now()}`);
    mkdirSync(directory, { recursive: true });
    tempDirs.push(directory);

    expect(parseSourceInput(directory)).toEqual({
      type: 'local',
      url: directory,
      localPath: directory,
    });
  });
});
