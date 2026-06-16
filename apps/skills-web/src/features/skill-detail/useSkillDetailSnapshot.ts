import {
  getInstallTarget,
  getSkillDetail,
  getSkillVersions,
  listGovernanceChanges,
} from '../../api/client';
import { useAsyncData } from '../../lib/useAsyncData';

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
  const governance = useAsyncData(
    () => listGovernanceChanges({ packageName }),
    [packageName, refreshKey]
  );

  const latestVersionItem = versionList.data?.items[0] ?? null;
  const installTargetVersionItem =
    installTarget.data && versionList.data?.items
      ? versionList.data.items.find((item) => item.version === installTarget.data?.version) ?? null
      : null;
  const statusTone: SkillDetailStatusTone = detail.data?.status === 'active' ? 'good' : 'bad';

  return {
    detail,
    versionList,
    installTarget,
    governance,
    latestVersionItem,
    installTargetVersionItem,
    statusTone,
    hasInstallTarget: Boolean(installTarget.data),
    installVersion:
      installTargetVersionItem?.version || installTarget.data?.version || '待确认版本',
  };
}
