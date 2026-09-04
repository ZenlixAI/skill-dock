package domain

import (
	"context"
	"time"
)

type SourceType string

const (
	SourceTypeNative          SourceType = "native"
	SourceTypeImported        SourceType = "imported"
	SourceTypeExternal        SourceType = "external"
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
	SourceKind          string      `json:"sourceKind,omitempty"`
	SourceLocator       string      `json:"sourceLocator,omitempty"`
	OriginRegistryURL   string      `json:"originRegistryUrl,omitempty"`
	OriginRepositoryURL string      `json:"originRepositoryUrl,omitempty"`
	OriginRevision      string      `json:"originRevision,omitempty"`
	ImportMode          string      `json:"importMode,omitempty"`
	LearnMoreURL        string      `json:"learnMoreUrl,omitempty"`
	Installable         bool        `json:"installable,omitempty"`
	Hosted              bool        `json:"hosted,omitempty"`
	Mirrored            bool        `json:"mirrored,omitempty"`
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
	PackageName         string        `json:"packageName"`
	Version             string        `json:"version"`
	VersionStatus       VersionStatus `json:"versionStatus"`
	InstallPolicy       InstallPolicy `json:"installPolicy"`
	PublishedAt         time.Time     `json:"publishedAt"`
	SourceKind          string        `json:"sourceKind,omitempty"`
	SourceLocator       string        `json:"sourceLocator,omitempty"`
	OriginRegistryURL   string        `json:"originRegistryUrl,omitempty"`
	OriginRepositoryURL string        `json:"originRepositoryUrl,omitempty"`
	OriginRevision      string        `json:"originRevision,omitempty"`
	ImportMode          string        `json:"importMode,omitempty"`
	Installable         bool          `json:"installable,omitempty"`
	Hosted              bool          `json:"hosted,omitempty"`
	Mirrored            bool          `json:"mirrored,omitempty"`
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
	RegistryAccessMode  string        `json:"registryAccessMode,omitempty"`
	SourceKind          string        `json:"sourceKind,omitempty"`
	SourceLocator       string        `json:"sourceLocator,omitempty"`
	OriginRegistryURL   string        `json:"originRegistryUrl,omitempty"`
	OriginRepositoryURL string        `json:"originRepositoryUrl,omitempty"`
	OriginRevision      string        `json:"originRevision,omitempty"`
	ImportMode          string        `json:"importMode,omitempty"`
	Installable         bool          `json:"installable,omitempty"`
	Hosted              bool          `json:"hosted,omitempty"`
	Mirrored            bool          `json:"mirrored,omitempty"`
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
	Query           string
	PackageName     string
	SourceType      string
	IncludeRetired  bool
	IncludeExternal bool
	Source          string
	Registry        string
	Scope           string
	Origin          string
	Sort            string
	AllowedPackages []string
	PageSize        int
	Cursor          int
}

type ExternalSearchQuery struct {
	Query    string
	PageSize int
	Registry string
	Scope    string
	Origin   string
}

type ExternalSkillSearcher interface {
	SearchSkills(ctx context.Context, query ExternalSearchQuery) (SkillsPage, error)
}

type CuratedSkillQuery struct {
	AllowedPackages []string
	PageSize        int
	Cursor          int
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

type SkillRecord struct {
	Detail         SkillDetail
	CurationReason string
	Versions       []SkillVersionDetail
}

type SampleCatalogData struct {
	Skills     []SkillRecord
	Governance []GovernanceRecord
	Artifacts  []ReleaseArtifact
}

type AgentSkillAggregate struct {
	Revision   uint64             `json:"revision"`
	Skill      SkillSnapshot      `json:"skill"`
	Governance []GovernanceRecord `json:"governance"`
}
