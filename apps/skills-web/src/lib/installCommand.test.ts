import { describe, expect, it } from 'vitest';
import {
  buildInstallCommand,
  getGovernanceActionLabel,
  getGovernanceTargetTypeLabel,
  getInstallPolicyLabel,
  getInstallRegistryLabel,
  getSkillStatusLabel,
  getSourceTypeLabel,
  getVersionStatusLabel,
} from './installCommand';

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

  it('uses a localized registry placeholder when registry metadata is missing', () => {
    expect(getInstallRegistryLabel('')).toBe('待补充 npm 仓库地址');
    expect(
      buildInstallCommand({
        packageName: '@platform.skill/agent-notes',
        version: '1.2.3',
        npmRegistry: '待补充 npm 仓库地址',
      })
    ).toBe('npm install @platform.skill/agent-notes@1.2.3 --registry=<待补充 npm 仓库地址>');
  });

  it('maps install policy values to Chinese labels', () => {
    expect(getInstallPolicyLabel('resolvable')).toBe('可直接安装');
    expect(getInstallPolicyLabel('catalog_only')).toBe('仅目录可见');
    expect(getInstallPolicyLabel('blocked')).toBe('已阻断安装');
  });

  it('maps status and governance values to Chinese labels', () => {
    expect(getSkillStatusLabel('active')).toBe('启用');
    expect(getSkillStatusLabel('retired')).toBe('已退役');
    expect(getSourceTypeLabel('native')).toBe('原生');
    expect(getSourceTypeLabel('imported')).toBe('导入');
    expect(getSourceTypeLabel('platform_created')).toBe('平台创建');
    expect(getVersionStatusLabel('published')).toBe('已发布');
    expect(getVersionStatusLabel('deprecated')).toBe('已弃用');
    expect(getVersionStatusLabel('yanked')).toBe('已撤回');
    expect(getGovernanceActionLabel('retire')).toBe('退役');
    expect(getGovernanceActionLabel('deprecate')).toBe('弃用');
    expect(getGovernanceActionLabel('yank')).toBe('撤回');
    expect(getGovernanceTargetTypeLabel('skill')).toBe('技能');
    expect(getGovernanceTargetTypeLabel('version')).toBe('版本');
  });
});
