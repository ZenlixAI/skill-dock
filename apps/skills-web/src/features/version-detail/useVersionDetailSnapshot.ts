import { getRegistryClientConfig, getVersionDetail, listGovernanceChanges } from '../../api/client';
import type { VersionStatus } from '../../api/types';
import { useAsyncData } from '../../lib/useAsyncData';
import { getInstallHostLabel, rewriteInstallTarballUrl } from '../../lib/installCommand';

type UseVersionDetailSnapshotOptions = {
  packageName: string;
  version: string;
  refreshKey: number;
};

type VersionDetailTone = 'good' | 'warn' | 'bad';

export function useVersionDetailSnapshot(options: UseVersionDetailSnapshotOptions) {
  const { packageName, version, refreshKey } = options;

  const detail = useAsyncData(() => getVersionDetail(packageName, version), [packageName, version, refreshKey]);
  const governance = useAsyncData(() => listGovernanceChanges({ packageName, version }), [packageName, version, refreshKey]);

  const currentVersionStatus: VersionStatus | 'unknown' = detail.data?.versionStatus ?? 'unknown';
  const versionTone: VersionDetailTone =
    currentVersionStatus === 'published' ? 'good' : currentVersionStatus === 'deprecated' ? 'warn' : 'bad';

  const registryConfig = getRegistryClientConfig();
  const publicInstallRegistry = registryConfig.baseUrl
    ? new URL('/repository/skills-npm/', registryConfig.baseUrl).toString()
    : '';
  const installRegistryHost = getInstallHostLabel(publicInstallRegistry);
  const installTarballUrl = rewriteInstallTarballUrl(detail.data?.registry.tarballUrl, publicInstallRegistry);

  return {
    detail,
    governance,
    currentVersionStatus,
    versionTone,
    installRegistryHost,
    installTarballUrl,
  };
}
