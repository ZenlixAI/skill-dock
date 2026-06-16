import { registryClientConfig } from './config';
import { catalogMutationClient } from './catalogMutationClient';
import { catalogReadClient, normalizeBundleMetadata, normalizeSkillsPage } from './catalogReadClient';
import { mockClient } from './mockClient';
import { normalizePackagingReview, packagingClient } from './packagingClient';
import { sessionClient } from './sessionClient';
import type {
  RegistryApiClient,
} from './types';
export {
  normalizeBundleMetadata,
  normalizePackagingReview,
  normalizeSkillsPage,
};

const realClient: RegistryApiClient = {
  ...sessionClient,
  ...catalogReadClient,
  ...packagingClient,
  ...catalogMutationClient,
};

export const apiClient: RegistryApiClient = registryClientConfig.mode === 'real' ? realClient : mockClient;

export const getRegistryClientConfig = () => apiClient.getConfig();
export const getSession = apiClient.getSession.bind(apiClient);
export const loginSession = apiClient.loginSession.bind(apiClient);
export const logoutSession = apiClient.logoutSession.bind(apiClient);

export const listSkills = apiClient.listSkills.bind(apiClient);
export const listCuratedSkills = apiClient.listCuratedSkills.bind(apiClient);
export const getSkillDetail = apiClient.getSkillDetail.bind(apiClient);
export const getSkillVersions = apiClient.getSkillVersions.bind(apiClient);
export const getInstallTarget = apiClient.getInstallTarget.bind(apiClient);
export const getVersionDetail = apiClient.getVersionDetail.bind(apiClient);
export const listGovernanceChanges = apiClient.listGovernanceChanges.bind(apiClient);
export const listAuditEvents = apiClient.listAuditEvents.bind(apiClient);
export const importReviewPackaging = apiClient.importReviewPackaging.bind(apiClient);
export const publishSkill = apiClient.publishSkill.bind(apiClient);
export const createReleaseRecord = apiClient.createReleaseRecord.bind(apiClient);
export const applySkillGovernanceChange = apiClient.applySkillGovernanceChange.bind(apiClient);
export const applyVersionGovernanceChange = apiClient.applyVersionGovernanceChange.bind(apiClient);
