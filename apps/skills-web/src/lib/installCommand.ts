import type {
  CatalogSourceType,
  GovernanceAction,
  GovernanceTargetType,
  InstallPolicy,
  SkillStatus,
  VersionStatus,
} from '../api/types';

type InstallCommandInput = {
  packageName: string;
  version: string;
  npmRegistry: string;
};

export function buildInstallCommand({ packageName, version, npmRegistry }: InstallCommandInput) {
  return `npm install ${packageName}@${version} --registry=${npmRegistry}`;
}

export function getInstallPolicyLabel(installPolicy: InstallPolicy) {
  if (installPolicy === 'resolvable') {
    return '可直接安装';
  }

  if (installPolicy === 'catalog_only') {
    return '仅目录可见';
  }

  return '已阻断安装';
}

export function getSkillStatusLabel(skillStatus: SkillStatus) {
  return skillStatus === 'active' ? '启用' : '已退役';
}

export function getSourceTypeLabel(sourceType: CatalogSourceType) {
  if (sourceType === 'native') {
    return '原生';
  }

  if (sourceType === 'imported') {
    return '导入';
  }

  return '平台创建';
}

export function getVersionStatusLabel(versionStatus: VersionStatus | 'unknown') {
  if (versionStatus === 'published') {
    return '已发布';
  }

  if (versionStatus === 'deprecated') {
    return '已弃用';
  }

  if (versionStatus === 'yanked') {
    return '已撤回';
  }

  return '未知';
}

export function getGovernanceActionLabel(action: GovernanceAction) {
  if (action === 'retire') {
    return '退役';
  }

  if (action === 'deprecate') {
    return '弃用';
  }

  return '撤回';
}

export function getGovernanceTargetTypeLabel(targetType: GovernanceTargetType) {
  return targetType === 'skill' ? '技能' : '版本';
}
