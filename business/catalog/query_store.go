package catalog

import "context"

type QueryStore interface {
	ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error)
	ListCuratedSkills(ctx context.Context, pageSize int, cursor int) (CuratedSkillsPage, error)
	GetSkill(ctx context.Context, packageName string) (SkillDetail, error)
	ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error)
	GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error)
	ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error)
	ListPackageVersions(ctx context.Context, packageName string) ([]SkillVersionDetail, error)
	ListPackageGovernance(ctx context.Context, packageName string, targetType GovernanceTargetType, version *string) ([]GovernanceRecord, error)
}
