import { describe, expect, it } from 'vitest';
import {
  buildInstallCommand,
  getGovernanceActionLabel,
  getGovernanceTargetTypeLabel,
  getInstallHostLabel,
  getInstallPolicyLabel,
  getInstallRegistryLabel,
  rewriteInstallTarballUrl,
  getSkillStatusLabel,
  getSourceTypeLabel,
  getVersionStatusLabel,
} from './installCommand';

describe('buildInstallCommand', () => {
  it('builds an npx install command pinned to version and registry env vars', () => {
    expect(
      buildInstallCommand({
        packageName: '@platform.skill/agent-notes',
        version: '1.2.3',
        registryHost: 'http://192.168.10.127:32080',
      })
    ).toBe(
      'SKILLS_REGISTRY_TOKEN=reader-token SKILLS_REGISTRY_URL=http://192.168.10.127:32080 npx --registry $SKILLS_REGISTRY_URL/repository/skills-npm/ skills add @platform.skill/agent-notes --version 1.2.3'
    );
  });

  it('uses a localized host placeholder when registry metadata is missing', () => {
    expect(getInstallHostLabel('')).toBe('待补充仓库 Host');
    expect(
      buildInstallCommand({
        packageName: '@platform.skill/agent-notes',
        version: '1.2.3',
        registryHost: '待补充仓库 Host',
      })
    ).toBe(
      'SKILLS_REGISTRY_TOKEN=reader-token SKILLS_REGISTRY_URL=<待补充仓库 Host> npx --registry $SKILLS_REGISTRY_URL/repository/skills-npm/ skills add @platform.skill/agent-notes --version 1.2.3'
    );
  });

  it('derives host labels from npm registry urls', () => {
    expect(getInstallRegistryLabel('http://192.168.10.127:32081/repository/skills-npm/')).toBe(
      'http://192.168.10.127:32081/repository/skills-npm/'
    );
    expect(getInstallHostLabel('http://192.168.10.127:32081/repository/skills-npm/')).toBe(
      'http://192.168.10.127:32081'
    );
  });

  it('rewrites tarball urls to use the registry host', () => {
    expect(
      rewriteInstallTarballUrl(
        'https://npm.internal.example.com/@platform.skill/agent-notes/-/agent-notes-1.2.3.tgz',
        'http://192.168.10.127:32080'
      )
    ).toBe('http://192.168.10.127:32080/@platform.skill/agent-notes/-/agent-notes-1.2.3.tgz');
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
