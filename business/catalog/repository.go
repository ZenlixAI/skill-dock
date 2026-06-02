package catalog

import "context"

type Repository interface {
	ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error)
	ListCuratedSkills(ctx context.Context, pageSize int, cursor int) (CuratedSkillsPage, error)
	GetSkill(ctx context.Context, packageName string) (SkillDetail, error)
	ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error)
	GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error)
	ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error)
	ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error)
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input ReleaseRecordInput) (ReleaseRecordResult, error)
	ApplySkillGovernanceChange(ctx context.Context, packageName string, input SkillGovernanceChangeInput, actor string) (GovernanceChangeResponse, error)
	ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input VersionGovernanceChangeInput, actor string) (GovernanceChangeResponse, error)
}
