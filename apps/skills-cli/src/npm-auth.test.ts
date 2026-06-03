import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { resolveNpmAuthHeaders, resolveNpmUserConfigPath } from './npm-auth.ts';

describe('npm auth helpers', () => {
  let testDir: string;
  let homeDir: string;

  beforeEach(() => {
    testDir = mkdtempSync(join(tmpdir(), 'skills-npm-auth-test-'));
    homeDir = mkdtempSync(join(tmpdir(), 'skills-npm-auth-home-'));
  });

  afterEach(() => {
    rmSync(testDir, { recursive: true, force: true });
    rmSync(homeDir, { recursive: true, force: true });
  });

  it('uses a matching scoped auth token from the cwd npmrc', () => {
    writeFileSync(
      join(testDir, '.npmrc'),
      '//npm.example.com/private/:_authToken=secret-token\n'
    );

    const result = resolveNpmAuthHeaders('https://npm.example.com/private/pkg/-/pkg-1.0.0.tgz', {
      cwd: testDir,
    });

    expect(result).toEqual({
      headers: {
        Authorization: 'Bearer secret-token',
      },
      userConfigPath: join(testDir, '.npmrc'),
    });
  });

  it('falls back to HOME .npmrc and decodes username/password auth', () => {
    writeFileSync(
      join(homeDir, '.npmrc'),
      '//npm.example.com/private/:username=publisher\n//npm.example.com/private/:_password=c2VjcmV0\n'
    );

    const result = resolveNpmAuthHeaders('https://npm.example.com/private/pkg/-/pkg-1.0.0.tgz', {
      cwd: testDir,
      env: { HOME: homeDir },
    });

    expect(result).toEqual({
      headers: {
        Authorization: `Basic ${Buffer.from('publisher:secret').toString('base64')}`,
      },
      userConfigPath: join(homeDir, '.npmrc'),
    });
  });

  it('prefers an explicit user config path over cwd discovery', () => {
    const explicitDir = join(testDir, 'config');
    mkdirSync(explicitDir, { recursive: true });
    const explicitPath = join(explicitDir, '.npmrc');
    writeFileSync(explicitPath, '//npm.example.com/private/:_authToken=explicit-token\n');
    writeFileSync(join(testDir, '.npmrc'), '//npm.example.com/private/:_authToken=cwd-token\n');

    expect(
      resolveNpmUserConfigPath({
        cwd: testDir,
        userConfigPath: explicitPath,
      })
    ).toBe(explicitPath);

    const result = resolveNpmAuthHeaders('https://npm.example.com/private/pkg/-/pkg-1.0.0.tgz', {
      cwd: testDir,
      userConfigPath: explicitPath,
    });

    expect(result.headers.Authorization).toBe('Bearer explicit-token');
  });

  it('returns empty headers when no matching auth entry exists', () => {
    writeFileSync(join(testDir, '.npmrc'), '//other.example.com/:_authToken=other-token\n');

    const result = resolveNpmAuthHeaders('https://npm.example.com/private/pkg/-/pkg-1.0.0.tgz', {
      cwd: testDir,
    });

    expect(result).toEqual({ headers: {} });
  });
});
