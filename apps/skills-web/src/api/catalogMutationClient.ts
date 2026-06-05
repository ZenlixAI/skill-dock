import { requestJSON } from './httpClient';
import type {
  GovernanceChangeResponse,
  RegistryApiClient,
  ReleaseRecordInput,
  ReleaseRecordResult,
  SkillGovernanceChangeInput,
  VersionGovernanceChangeInput,
} from './types';

export const catalogMutationClient: Pick<
  RegistryApiClient,
  'createReleaseRecord' | 'applySkillGovernanceChange' | 'applyVersionGovernanceChange'
> = {
  createReleaseRecord(packageName: string, version: string, input: ReleaseRecordInput) {
    return requestJSON<ReleaseRecordResult>(
      `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${encodeURIComponent(version)}/release-record`,
      {
        method: 'PUT',
        body: JSON.stringify(input),
      }
    );
  },

  applySkillGovernanceChange(packageName: string, input: SkillGovernanceChangeInput) {
    return requestJSON<GovernanceChangeResponse>(
      `/api/v1/skills/${encodeURIComponent(packageName)}/governance-changes`,
      {
        method: 'POST',
        body: JSON.stringify(input),
      }
    );
  },

  applyVersionGovernanceChange(
    packageName: string,
    version: string,
    input: VersionGovernanceChangeInput
  ) {
    return requestJSON<GovernanceChangeResponse>(
      `/api/v1/skills/${encodeURIComponent(packageName)}/versions/${encodeURIComponent(version)}/governance-changes`,
      {
        method: 'POST',
        body: JSON.stringify(input),
      }
    );
  },
};
