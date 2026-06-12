import type {
  CatalogSourceType,
  GovernanceAction,
  GovernanceTargetType,
  InstallPolicy,
  SkillStatus,
  VersionStatus,
} from '../api/types';

type InstallCommandInput = {
  registryBaseUrl: string;
  packageName: string;
  version: string;
};

const INSTALL_REGISTRY_FALLBACK_LABEL = '待补充 npm 仓库地址';
const INSTALL_HOST_FALLBACK_LABEL = '待补充仓库 Host';
function parseUrl(value?: string | null): URL | null {
  const trimmedValue = value?.trim();
  if (!trimmedValue) {
    return null;
  }

  try {
    return new URL(trimmedValue);
  } catch {
    return null;
  }
}

export function getInstallRegistryLabel(npmRegistry?: string | null) {
  const trimmedRegistry = npmRegistry?.trim();

  return trimmedRegistry ? trimmedRegistry : INSTALL_REGISTRY_FALLBACK_LABEL;
}

export function getInstallHostLabel(npmRegistry?: string | null) {
  const parsedRegistry = parseUrl(npmRegistry);
  if (parsedRegistry) {
    return parsedRegistry.origin;
  }

  const trimmedRegistry = npmRegistry?.trim();
  return trimmedRegistry ? trimmedRegistry.replace(/\/+$/, '') : INSTALL_HOST_FALLBACK_LABEL;
}

export function rewriteInstallTarballUrl(tarballUrl?: string | null, registryHost?: string | null) {
  const parsedTarballUrl = parseUrl(tarballUrl);
  const parsedRegistryHost = parseUrl(registryHost);

  if (!parsedTarballUrl || !parsedRegistryHost) {
    return tarballUrl?.trim() || '';
  }

  return `${parsedRegistryHost.origin}${parsedTarballUrl.pathname}${parsedTarballUrl.search}${parsedTarballUrl.hash}`;
}

export function buildInstallCommand({ registryBaseUrl, packageName, version }: InstallCommandInput) {
  const normalizedRegistryBaseUrl = registryBaseUrl.trim() || INSTALL_REGISTRY_FALLBACK_LABEL;
  return `npx --registry ${normalizedRegistryBaseUrl} skills add ${packageName} --version ${version}`;
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
