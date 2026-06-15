package catalog

import internaldomain "skill_registry/internal/catalog/domain"

type SourceType = internaldomain.SourceType

const (
	SourceTypeNative          = internaldomain.SourceTypeNative
	SourceTypeImported        = internaldomain.SourceTypeImported
	SourceTypePlatformCreated = internaldomain.SourceTypePlatformCreated
)

type SkillStatus = internaldomain.SkillStatus

const (
	SkillStatusActive  = internaldomain.SkillStatusActive
	SkillStatusRetired = internaldomain.SkillStatusRetired
)

type VersionStatus = internaldomain.VersionStatus

const (
	VersionStatusPublished  = internaldomain.VersionStatusPublished
	VersionStatusDeprecated = internaldomain.VersionStatusDeprecated
	VersionStatusYanked     = internaldomain.VersionStatusYanked
)

type InstallPolicy = internaldomain.InstallPolicy

const (
	InstallPolicyCatalogOnly = internaldomain.InstallPolicyCatalogOnly
	InstallPolicyResolvable  = internaldomain.InstallPolicyResolvable
	InstallPolicyBlocked     = internaldomain.InstallPolicyBlocked
)

type GovernanceTargetType = internaldomain.GovernanceTargetType

const (
	GovernanceTargetSkill   = internaldomain.GovernanceTargetSkill
	GovernanceTargetVersion = internaldomain.GovernanceTargetVersion
)

type GovernanceAction = internaldomain.GovernanceAction

const (
	GovernanceActionRetire    = internaldomain.GovernanceActionRetire
	GovernanceActionDeprecate = internaldomain.GovernanceActionDeprecate
	GovernanceActionYank      = internaldomain.GovernanceActionYank
)

type SkillSummary = internaldomain.SkillSummary
type SkillDetail = internaldomain.SkillDetail
type CuratedSkillSummary = internaldomain.CuratedSkillSummary
type RegistryMetadata = internaldomain.RegistryMetadata
type BundleMetadata = internaldomain.BundleMetadata
type SkillVersionSummary = internaldomain.SkillVersionSummary
type SkillVersionDetail = internaldomain.SkillVersionDetail
type SkillSnapshot = internaldomain.SkillSnapshot
type GovernanceRecord = internaldomain.GovernanceRecord
type InstallResolution = internaldomain.InstallResolution
type ReleaseRecordResult = internaldomain.ReleaseRecordResult
type ReleaseRecordInput = internaldomain.ReleaseRecordInput
type SkillGovernanceChangeInput = internaldomain.SkillGovernanceChangeInput
type VersionGovernanceChangeInput = internaldomain.VersionGovernanceChangeInput
type GovernanceChangeResultingState = internaldomain.GovernanceChangeResultingState
type GovernanceChangeResponse = internaldomain.GovernanceChangeResponse
type PageInfo = internaldomain.PageInfo
type SkillQuery = internaldomain.SkillQuery
type CuratedSkillQuery = internaldomain.CuratedSkillQuery
type VersionQuery = internaldomain.VersionQuery
type GovernanceQuery = internaldomain.GovernanceQuery
type SkillsPage = internaldomain.SkillsPage
type CuratedSkillsPage = internaldomain.CuratedSkillsPage
type VersionsPage = internaldomain.VersionsPage
type GovernancePage = internaldomain.GovernancePage
type ReleaseArtifact = internaldomain.ReleaseArtifact
