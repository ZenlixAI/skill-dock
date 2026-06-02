package catalog

import "time"

type SourceType string

const (
	SourceTypeNative          SourceType = "native"
	SourceTypeImported        SourceType = "imported"
	SourceTypePlatformCreated SourceType = "platform_created"
)

type SkillStatus string

const (
	SkillStatusActive  SkillStatus = "active"
	SkillStatusRetired SkillStatus = "retired"
)

type VersionStatus string

const (
	VersionStatusPublished  VersionStatus = "published"
	VersionStatusDeprecated VersionStatus = "deprecated"
	VersionStatusYanked     VersionStatus = "yanked"
)

type InstallPolicy string

const (
	InstallPolicyCatalogOnly InstallPolicy = "catalog_only"
	InstallPolicyResolvable  InstallPolicy = "resolvable"
	InstallPolicyBlocked     InstallPolicy = "blocked"
)

type GovernanceTargetType string

const (
	GovernanceTargetSkill   GovernanceTargetType = "skill"
	GovernanceTargetVersion GovernanceTargetType = "version"
)

type GovernanceAction string

const (
	GovernanceActionRetire    GovernanceAction = "retire"
	GovernanceActionDeprecate GovernanceAction = "deprecate"
	GovernanceActionYank      GovernanceAction = "yank"
)

type SkillSummary struct {
	PackageName         string      `json:"packageName"`
	DisplayName         string      `json:"displayName"`
	Description         string      `json:"description"`
	SourceType          SourceType  `json:"sourceType"`
	Status              SkillStatus `json:"status"`
	LatestVersion       *string     `json:"latestVersion"`
	LatestVersionStatus *string     `json:"latestVersionStatus"`
	UpdatedAt           time.Time   `json:"updatedAt"`
}

type SkillDetail struct {
	SkillSummary
	CreatedAt time.Time `json:"createdAt"`
}

type CuratedSkillSummary struct {
	SkillSummary
	CurationReason string `json:"curationReason"`
}

type RegistryMetadata struct {
	NPMRegistry string    `json:"npmRegistry"`
	TarballURL  string    `json:"tarballUrl"`
	Integrity   string    `json:"integrity"`
	Shasum      string    `json:"shasum"`
	PublishedAt time.Time `json:"publishedAt"`
	DistTags    []string  `json:"distTags,omitempty"`
}

type BundleMetadata struct {
	SkillManifestDigest string   `json:"skillManifestDigest"`
	Components          []string `json:"components"`
}

type SkillVersionSummary struct {
	PackageName   string        `json:"packageName"`
	Version       string        `json:"version"`
	VersionStatus VersionStatus `json:"versionStatus"`
	InstallPolicy InstallPolicy `json:"installPolicy"`
	PublishedAt   time.Time     `json:"publishedAt"`
}

type SkillVersionDetail struct {
	SkillVersionSummary
	Registry RegistryMetadata `json:"registry"`
	Bundle   BundleMetadata   `json:"bundle"`
}

type SkillSnapshot struct {
	Detail         SkillDetail          `json:"detail"`
	CurationReason string               `json:"curationReason,omitempty"`
	Versions       []SkillVersionDetail `json:"versions"`
}

type GovernanceRecord struct {
	ID                        string               `json:"id"`
	TargetType                GovernanceTargetType `json:"targetType"`
	PackageName               string               `json:"packageName"`
	Version                   *string              `json:"version,omitempty"`
	Action                    GovernanceAction     `json:"action"`
	Reason                    string               `json:"reason"`
	ReplacementRecommendation *string              `json:"replacementRecommendation,omitempty"`
	ChangedBy                 string               `json:"changedBy"`
	ChangedAt                 time.Time            `json:"changedAt"`
}

type InstallResolution struct {
	PackageName         string        `json:"packageName"`
	Version             string        `json:"version"`
	NPMRegistry         string        `json:"npmRegistry"`
	Integrity           string        `json:"integrity"`
	TarballURL          string        `json:"tarballUrl"`
	SkillManifestDigest string        `json:"skillManifestDigest"`
	VersionStatus       VersionStatus `json:"versionStatus"`
}

type ReleaseRecordResult struct {
	PackageName      string        `json:"packageName"`
	Version          string        `json:"version"`
	Created          bool          `json:"created"`
	IdempotentReplay bool          `json:"idempotentReplay"`
	VersionStatus    VersionStatus `json:"versionStatus"`
	InstallPolicy    InstallPolicy `json:"installPolicy"`
}

type ReleaseRecordInput struct {
	PublishRegistry *string  `json:"publishRegistry,omitempty"`
	ManifestDigest  *string  `json:"manifestDigest,omitempty"`
	Components      []string `json:"components,omitempty"`
}

type SkillGovernanceChangeInput struct {
	Action GovernanceAction `json:"action"`
	Reason string           `json:"reason"`
}

type VersionGovernanceChangeInput struct {
	Action                    GovernanceAction `json:"action"`
	Reason                    string           `json:"reason"`
	ReplacementRecommendation *string          `json:"replacementRecommendation,omitempty"`
}

type GovernanceChangeResultingState struct {
	Status        *SkillStatus   `json:"status,omitempty"`
	VersionStatus *VersionStatus `json:"versionStatus,omitempty"`
}

type GovernanceChangeResponse struct {
	Record         GovernanceRecord               `json:"record"`
	ResultingState GovernanceChangeResultingState `json:"resultingState"`
}

type PageInfo struct {
	NextCursor  *string `json:"nextCursor"`
	HasNextPage bool    `json:"hasNextPage"`
}

type SkillQuery struct {
	Query          string
	PackageName    string
	SourceType     string
	IncludeRetired bool
	Sort           string
	PageSize       int
	Cursor         int
}

type VersionQuery struct {
	PackageName       string
	IncludeDeprecated bool
	IncludeYanked     bool
	PageSize          int
	Cursor            int
}

type GovernanceQuery struct {
	PackageName string
	Version     string
	TargetType  string
	PageSize    int
	Cursor      int
}

type SkillsPage struct {
	Items    []SkillSummary `json:"items"`
	PageInfo PageInfo       `json:"pageInfo"`
}

type CuratedSkillsPage struct {
	Items    []CuratedSkillSummary `json:"items"`
	PageInfo PageInfo              `json:"pageInfo"`
}

type VersionsPage struct {
	Items    []SkillVersionSummary `json:"items"`
	PageInfo PageInfo              `json:"pageInfo"`
}

type GovernancePage struct {
	Items    []GovernanceRecord `json:"items"`
	PageInfo PageInfo           `json:"pageInfo"`
}

type ReleaseArtifact struct {
	SkillDisplayName string             `json:"skillDisplayName"`
	SkillDescription string             `json:"skillDescription"`
	SkillSourceType  SourceType         `json:"skillSourceType"`
	Version          SkillVersionDetail `json:"version"`
	ValidationError  string             `json:"validationError,omitempty"`
	RegistryDown     bool               `json:"registryDown,omitempty"`
}
