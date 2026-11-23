import { mkdtempSync, rmSync, statSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { afterEach, describe, expect, it } from 'vitest';
import { clearCredentials, credentialsPath, readCredentials, writeCredentials } from './credentials.ts';

const directories: string[] = [];

afterEach(() => {
  for (const directory of directories.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

describe('credentials', () => {
  it('writes, reads, and clears the local credential with restrictive permissions', () => {
    const directory = mkdtempSync(join(tmpdir(), 'skill-dock-credentials-'));
    directories.push(directory);
    const path = join(directory, 'credentials.json');

    writeCredentials(
      {
        logtoEndpoint: 'https://logto.example.com',
        clientId: 'cli',
        refreshToken: 'refresh',
        accessToken: 'access',
        accessTokenExpiresAt: Date.now() + 60_000,
      },
      { path }
    );

    const stored = readCredentials({ path });
    expect(stored?.refreshToken).toBe('refresh');
    expect(statSync(path).mode & 0o777).toBe(0o600);
    expect(credentialsPath({ path })).toBe(path);

    clearCredentials({ path });
    expect(readCredentials({ path })).toBeNull();
  });
});
