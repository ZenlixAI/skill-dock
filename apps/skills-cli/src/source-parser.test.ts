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
      url: 'https://example.com/org/repo.git',
      ref: 'v1',
    });
  });

  it('parses GitLab.com repository URLs and clones over SSH', () => {
    expect(parseSourceInput('https://gitlab.com/group/repo')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:group/repo.git',
    });
  });

  it('parses GitLab subgroup and .git repository URLs', () => {
    expect(parseSourceInput('https://gitlab.com/group/subgroup/repo.git')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:group/subgroup/repo.git',
    });
  });

  it('parses GitLab tree URLs with ref and subpath', () => {
    expect(parseSourceInput('https://gitlab.com/group/repo/-/tree/main/skills/pdf')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:group/repo.git',
      ref: 'main',
      subpath: 'skills/pdf',
    });
  });

  it('parses self-hosted GitLab tree URLs', () => {
    expect(parseSourceInput('https://gitlab.example.com/group/repo/-/tree/v1.0')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.example.com:group/repo.git',
      ref: 'v1.0',
    });
  });

  it('parses GitLab query selectors', () => {
    expect(parseSourceInput('https://gitlab.com/group/repo?ref=main&path=skills/pdf')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:group/repo.git',
      subpath: 'skills/pdf',
      ref: 'main',
    });
  });

  it('parses the gitlab: shorthand and preserves the skill filter', () => {
    expect(parseSourceInput('gitlab:group/subgroup/repo@pdf')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:group/subgroup/repo.git',
      skillFilter: 'pdf',
    });
  });

  it('parses host-qualified GitLab shorthand', () => {
    expect(parseSourceInput('gitlab.com/group/repo')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:group/repo.git',
    });
    expect(parseSourceInput('gitlab.example.com/group/subgroup/repo')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.example.com:group/subgroup/repo.git',
    });
  });

  it('parses GitLab SSH URLs as first-class GitLab sources', () => {
    expect(parseSourceInput('git@gitlab.com:group/repo.git')).toEqual({
      type: 'gitlab',
      url: 'git@gitlab.com:group/repo.git',
    });
    expect(parseSourceInput('ssh://git@gitlab.example.com:2222/group/repo.git')).toEqual({
      type: 'gitlab',
      url: 'ssh://git@gitlab.example.com:2222/group/repo.git',
    });
  });

  it('rejects unsupported GitLab UI paths', () => {
    expect(parseSourceInput('https://gitlab.com/group/repo/-/blob/main/SKILL.md')).toBeNull();
  });

  it('does not treat other hosts as GitLab', () => {
    expect(parseSourceInput('https://notgitlab.com/group/repo')).toBeNull();
    expect(parseSourceInput('https://mygitlab.com/group/repo')).toBeNull();
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
