import { existsSync, mkdirSync, rmSync, symlinkSync, writeFileSync } from 'fs';
import { spawnSync } from 'child_process';
import { tmpdir } from 'os';
import { dirname, join } from 'path';
import { afterEach, describe, expect, it } from 'vitest';
import { packLocalSkillDirectory, validateLocalWorkspace } from './local-packaging.ts';

function createTempSkillDir(name: string): string {
  const directory = join(tmpdir(), `skills-local-packaging-${name}-${Date.now()}-${Math.random()}`);
  mkdirSync(directory, { recursive: true });
  return directory;
}

function writeBaseSkill(directory: string): void {
  writeFileSync(
    join(directory, 'SKILL.md'),
    ['---', 'name: demo-skill', 'description: Demo skill', '---', '', 'Body'].join('\n')
  );
  writeFileSync(
    join(directory, 'package.json'),
    JSON.stringify(
      {
        name: '@acme.skill/demo-skill',
        version: '0.1.0',
        description: 'Demo skill',
      },
      null,
      2
    )
  );
  mkdirSync(join(directory, 'commands'), { recursive: true });
  writeFileSync(join(directory, 'commands', 'run.md'), '# run');
}

describe('local packaging', () => {
  const tempDirectories: string[] = [];

  afterEach(() => {
    for (const directory of tempDirectories.splice(0)) {
      if (existsSync(directory)) {
        rmSync(directory, { recursive: true, force: true });
      }
    }
  });

  it('ignores packaging scratch paths during local validation', async () => {
    const directory = createTempSkillDir('validate');
    tempDirectories.push(directory);
    writeBaseSkill(directory);

    mkdirSync(join(directory, '.dist'), { recursive: true });
    writeFileSync(join(directory, '.dist', 'SKILL.md'), 'ignored nested skill');
    mkdirSync(join(directory, '.git'), { recursive: true });
    writeFileSync(join(directory, '.git', 'config'), 'ignored git config');
    writeFileSync(
      join(directory, '.skill-packaging-import-report.json'),
      JSON.stringify({ ignored: true })
    );

    await expect(validateLocalWorkspace(directory)).resolves.toEqual([]);
  });

  it('accepts a SKILL.md-only workspace as valid capability content', async () => {
    const directory = createTempSkillDir('skill-md-only');
    tempDirectories.push(directory);
    writeFileSync(
      join(directory, 'SKILL.md'),
      ['---', 'name: demo-skill', 'description: Demo skill', '---', '', 'Body'].join('\n')
    );
    writeFileSync(
      join(directory, 'package.json'),
      JSON.stringify(
        {
          name: '@acme.skill/demo-skill',
          version: '0.1.0',
          description: 'Demo skill',
        },
        null,
        2
      )
    );

    await expect(validateLocalWorkspace(directory)).resolves.toEqual([]);
  });

  it('excludes .dist, .git, and import reports from the packed bundle', async () => {
    const directory = createTempSkillDir('pack');
    tempDirectories.push(directory);
    writeBaseSkill(directory);

    mkdirSync(join(directory, '.dist'), { recursive: true });
    writeFileSync(join(directory, '.dist', 'bundle.tgz'), 'ignored bundle');
    mkdirSync(join(directory, '.git'), { recursive: true });
    writeFileSync(join(directory, '.git', 'HEAD'), 'ref: refs/heads/main\n');
    writeFileSync(
      join(directory, '.skill-packaging-import-report.json'),
      JSON.stringify({ ignored: true })
    );

    const result = await packLocalSkillDirectory({ directory });
    tempDirectories.push(dirname(dirname(result.bundlePath)));

    const listing = spawnSync('tar', ['-tzf', result.bundlePath], {
      encoding: 'utf-8',
      shell: process.platform === 'win32',
    });

    expect(listing.status).toBe(0);
    expect(listing.stdout).toContain('package/SKILL.md');
    expect(listing.stdout).toContain('package/package.json');
    expect(listing.stdout).toContain('package/commands/run.md');
    expect(listing.stdout).not.toContain('package/.dist/bundle.tgz');
    expect(listing.stdout).not.toContain('package/.git/HEAD');
    expect(listing.stdout).not.toContain('package/.skill-packaging-import-report.json');

    result.cleanup();
  });

  it('excludes macOS metadata sidecars from the packed bundle', async () => {
    const directory = createTempSkillDir('mac-metadata');
    tempDirectories.push(directory);
    writeBaseSkill(directory);

    writeFileSync(join(directory, '._SKILL.md'), 'appledouble');
    writeFileSync(join(directory, '.DS_Store'), 'finder');
    writeFileSync(join(directory, 'commands', '._run.md'), 'appledouble');

    const result = await packLocalSkillDirectory({ directory });
    tempDirectories.push(dirname(dirname(result.bundlePath)));

    const listing = spawnSync('tar', ['-tzf', result.bundlePath], {
      encoding: 'utf-8',
      shell: process.platform === 'win32',
    });

    expect(listing.status).toBe(0);
    expect(listing.stdout).toContain('package/SKILL.md');
    expect(listing.stdout).toContain('package/commands/run.md');
    expect(listing.stdout).not.toContain('package/._SKILL.md');
    expect(listing.stdout).not.toContain('package/.DS_Store');
    expect(listing.stdout).not.toContain('package/commands/._run.md');

    result.cleanup();
  });

  it('rejects symbolic links during local validation', async () => {
    const directory = createTempSkillDir('symlink');
    tempDirectories.push(directory);
    writeBaseSkill(directory);

    writeFileSync(join(directory, 'outside.txt'), 'outside');
    symlinkSync(join(directory, 'outside.txt'), join(directory, 'commands', 'linked.txt'));

    await expect(validateLocalWorkspace(directory)).resolves.toEqual(['unsafe_path_entry']);
  });
});
