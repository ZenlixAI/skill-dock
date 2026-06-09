import type { InstallPolicy } from '../api/types';

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
