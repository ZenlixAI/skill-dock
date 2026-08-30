import { getRegistryClientConfig, getInstallTarget, getSkillDetail, getSkillVersions, listGovernanceChanges } from '../../api/client';
import { useAsyncData } from '../../lib/useAsyncData';
import { getInstallHostLabel, rewriteInstallTarballUrl } from '../../lib/installCommand';

type UseSkillDetailSnapshotOptions = {
  packageName: string;
  refreshKey: number;
};

type SkillDetailStatusTone = 'good' | 'bad';

export function useSkillDetailSnapshot(options: UseSkillDetailSnapshotOptions) {
  const { packageName, refreshKey } = options;

  const detail = useAsyncData(() => getSkillDetail(packageName), [packageName, refreshKey]);
  const versionList = useAsyncData(() => getSkillVersions(packageName), [packageName, refreshKey]);
  const installTarget = useAsyncData(() => getInstallTarget(packageName), [packageName, refreshKey]);
  const governance = useAsyncData(() => listGovernanceChanges({ packageName }), [packageName, refreshKey]);

  const latestVersionItem = versionList.data?.items[0] ?? null;
  const installTargetData = installTarget.data;
  const installTargetVersionItem =
    installTargetData && versionList.data?.items
      ? versionList.data.items.find((item) => item.version === installTargetData.version) ?? null
      : null;
  const statusTone: SkillDetailStatusTone = detail.data?.status === 'active' ? 'good' : 'bad';

  const registryConfig = getRegistryClientConfig();
  const publicInstallRegistry = registryConfig.baseUrl
    ? new URL('/repository/skills-npm/', registryConfig.baseUrl).toString()
    : '';
  const installRegistryHost = getInstallHostLabel(publicInstallRegistry);
  const installTarballUrl = rewriteInstallTarballUrl(installTargetData?.tarballUrl, publicInstallRegistry);

  return {
    detail,
    versionList,
    installTarget,
    governance,
    latestVersionItem,
    installTargetVersionItem,
    statusTone,
    installVersion: installTargetVersionItem?.version || installTargetData?.version || '待确认版本',
    installRegistryHost,
    installTarballUrl,
  };
}
