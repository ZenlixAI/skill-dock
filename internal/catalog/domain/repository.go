package domain

import "context"

type QueryStore interface {
	ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error)
	ListCuratedSkills(ctx context.Context, query CuratedSkillQuery) (CuratedSkillsPage, error)
	GetSkill(ctx context.Context, packageName string) (SkillDetail, error)
	ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error)
	GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error)
	GetLatestResolvableVersion(ctx context.Context, packageName string) (SkillVersionDetail, error)
	ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error)
	ListPackageVersions(ctx context.Context, packageName string) ([]SkillVersionDetail, error)
	ListPackageGovernance(ctx context.Context, packageName string, targetType GovernanceTargetType, version *string) ([]GovernanceRecord, error)
}

type AggregateStore interface {
	LoadAggregates(ctx context.Context) ([]AgentSkillAggregate, error)
	SaveAggregates(ctx context.Context, aggregates []AgentSkillAggregate) error
}

type AggregateStateStore interface {
	IsEmpty(ctx context.Context) (bool, error)
	GetAggregate(ctx context.Context, packageName string) (AgentSkillAggregate, error)
	PutAggregate(ctx context.Context, aggregate AgentSkillAggregate, expectedRevision uint64) error
	Ping(ctx context.Context) error
}

type CatalogStateStore interface {
	QueryStore
	AggregateStateStore
}

type ArtifactSource interface {
	FindReleaseArtifact(ctx context.Context, packageName string, version string) (ReleaseArtifact, error)
}

type ReadRepository interface {
	ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error)
	ListCuratedSkills(ctx context.Context, query CuratedSkillQuery) (CuratedSkillsPage, error)
	GetSkill(ctx context.Context, packageName string) (SkillDetail, error)
	ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error)
	GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error)
	ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error)
	ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error)
}

type WriteRepository interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input ReleaseRecordInput) (ReleaseRecordResult, error)
	ApplySkillGovernanceChange(ctx context.Context, packageName string, input SkillGovernanceChangeInput, actor string) (GovernanceChangeResponse, error)
	ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input VersionGovernanceChangeInput, actor string) (GovernanceChangeResponse, error)
}

type Repository interface {
	ReadRepository
	WriteRepository
}
