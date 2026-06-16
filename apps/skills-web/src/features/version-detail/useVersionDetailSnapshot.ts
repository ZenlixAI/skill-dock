import { getVersionDetail, listGovernanceChanges } from '../../api/client';
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
  const governance = useAsyncData(
    () => listGovernanceChanges({ packageName, version }),
    [packageName, version, refreshKey]
  );

  const currentVersionStatus: VersionStatus | 'unknown' = detail.data?.versionStatus ?? 'unknown';
  const versionTone: VersionDetailTone =
    currentVersionStatus === 'published' ? 'good' : currentVersionStatus === 'deprecated' ? 'warn' : 'bad';
  const installRegistryHost = getInstallHostLabel(detail.data?.registry.npmRegistry);
  const installTarballUrl = rewriteInstallTarballUrl(
    detail.data?.registry.tarballUrl,
    detail.data?.registry.npmRegistry
  );

  return {
    detail,
    governance,
    currentVersionStatus,
    versionTone,
    installRegistryHost,
    installTarballUrl,
  };
}
