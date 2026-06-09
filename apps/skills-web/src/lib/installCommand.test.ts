import { describe, expect, it } from 'vitest';
import { buildInstallCommand, getInstallPolicyLabel } from './installCommand';

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

  it('maps install policy values to Chinese labels', () => {
    expect(getInstallPolicyLabel('resolvable')).toBe('可直接安装');
    expect(getInstallPolicyLabel('catalog_only')).toBe('仅目录可见');
    expect(getInstallPolicyLabel('blocked')).toBe('已阻断安装');
  });
});
