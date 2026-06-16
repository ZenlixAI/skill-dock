import { listAuditEvents, listGovernanceChanges, listSkills } from '../../api/client';
import { useAsyncData } from '../../lib/useAsyncData';

type UseWorkbenchSnapshotOptions = {
  pageSize?: number;
};

export function useWorkbenchSnapshot(options: UseWorkbenchSnapshotOptions = {}) {
  const pageSize = options.pageSize ?? 8;
  const skills = useAsyncData(() => listSkills({ includeRetired: true, pageSize }), [pageSize]);
  const governance = useAsyncData(() => listGovernanceChanges({ pageSize }), [pageSize]);
  const audit = useAsyncData(() => listAuditEvents({ pageSize }), [pageSize]);

  const packagingEvents = audit.data?.items.filter((item) => item.action.startsWith('packaging.')) || [];
  const releaseEvents = audit.data?.items.filter((item) => item.action === 'release.record') || [];

  return {
    skills,
    governance,
    audit,
    packagingEvents,
    releaseEvents,
  };
}
