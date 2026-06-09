import { describe, expect, it } from 'vitest';
import { buildInstallCommand } from './installCommand';

describe('buildInstallCommand', () => {
  it('builds an npm install command pinned to version and registry', () => {
    expect(
      buildInstallCommand({
        packageName: '@platform.skill/agent-notes',
        version: '1.2.3',
        npmRegistry: 'https://npm.internal.example.com',
      })
    ).toBe('npm install @platform.skill/agent-notes@1.2.3 --registry=https://npm.internal.example.com');
  });
});
