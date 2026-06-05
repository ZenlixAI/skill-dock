package catalog

import (
	"context"
	"testing"
)

type readModuleQueryStoreStub struct {
	skills             map[string]SkillDetail
	versions           map[string][]SkillVersionDetail
	governance         map[string][]GovernanceRecord
	getSkillCalls      int
	listPackageVersion struct {
		calls       int
		packageName string
	}
	listPackageGovernance struct {
		calls       int
		packageName string
		targetType  GovernanceTargetType
		version     *string
	}
}

var _ QueryStore = (*readModuleQueryStoreStub)(nil)

func newReadModuleQueryStoreStub(data SampleCatalogData) *readModuleQueryStoreStub {
	skills := make(map[string]SkillDetail, len(data.Skills))
	versions := make(map[string][]SkillVersionDetail, len(data.Skills))
	governance := make(map[string][]GovernanceRecord)

	for _, skill := range data.Skills {
		skills[skill.Detail.PackageName] = skill.Detail
		versions[skill.Detail.PackageName] = append([]SkillVersionDetail(nil), skill.Versions...)
	}
	for _, record := range data.Governance {
		key := governanceKey(record.PackageName, record.TargetType, record.Version)
		governance[key] = append(governance[key], record)
	}

	return &readModuleQueryStoreStub{
		skills:     skills,
		versions:   versions,
		governance: governance,
	}
}

func (s *readModuleQueryStoreStub) ListSkills(context.Context, SkillQuery) (SkillsPage, error) {
	panic("unexpected call to ListSkills")
}

func (s *readModuleQueryStoreStub) ListCuratedSkills(context.Context, int, int) (CuratedSkillsPage, error) {
	panic("unexpected call to ListCuratedSkills")
}

func (s *readModuleQueryStoreStub) GetSkill(_ context.Context, packageName string) (SkillDetail, error) {
	s.getSkillCalls++
	skill, ok := s.skills[packageName]
	if !ok {
		return SkillDetail{}, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return skill, nil
}

func (s *readModuleQueryStoreStub) ListVersions(context.Context, VersionQuery) (VersionsPage, error) {
	panic("unexpected call to ListVersions")
}

func (s *readModuleQueryStoreStub) GetVersion(context.Context, string, string) (SkillVersionDetail, error) {
	panic("unexpected call to GetVersion")
}

func (s *readModuleQueryStoreStub) ListGovernanceChanges(context.Context, GovernanceQuery) (GovernancePage, error) {
	panic("unexpected call to ListGovernanceChanges")
}

func (s *readModuleQueryStoreStub) ListPackageVersions(_ context.Context, packageName string) ([]SkillVersionDetail, error) {
	s.listPackageVersion.calls++
	s.listPackageVersion.packageName = packageName

	versions, ok := s.versions[packageName]
	if !ok {
		return nil, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return append([]SkillVersionDetail(nil), versions...), nil
}

func (s *readModuleQueryStoreStub) ListPackageGovernance(_ context.Context, packageName string, targetType GovernanceTargetType, version *string) ([]GovernanceRecord, error) {
	s.listPackageGovernance.calls++
	s.listPackageGovernance.packageName = packageName
	s.listPackageGovernance.targetType = targetType
	s.listPackageGovernance.version = cloneStringPtr(version)

	if _, ok := s.skills[packageName]; !ok {
		return nil, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	key := governanceKey(packageName, targetType, version)
	return append([]GovernanceRecord(nil), s.governance[key]...), nil
}

func governanceKey(packageName string, targetType GovernanceTargetType, version *string) string {
	if version == nil {
		return packageName + "|" + string(targetType) + "|"
	}
	return packageName + "|" + string(targetType) + "|" + *version
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func TestReadModuleResolvesInstallTargetFromQueryableState(t *testing.T) {
	store := newReadModuleQueryStoreStub(SampleData())
	module := newReadModule(store)

	resolution, err := module.ResolveInstallTarget(context.Background(), "@platform.skill/pdf-skill", "")
	if err != nil {
		t.Fatalf("resolve install target: %v", err)
	}
	if resolution.PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("unexpected package %q", resolution.PackageName)
	}
	if resolution.Version != "1.2.0" {
		t.Fatalf("unexpected version %q", resolution.Version)
	}
	if store.getSkillCalls != 1 {
		t.Fatalf("expected one GetSkill call, got %d", store.getSkillCalls)
	}
	if store.listPackageVersion.calls != 1 {
		t.Fatalf("expected one package-version projection call, got %d", store.listPackageVersion.calls)
	}
	if store.listPackageVersion.packageName != "@platform.skill/pdf-skill" {
		t.Fatalf("unexpected package-version lookup %q", store.listPackageVersion.packageName)
	}
	if store.listPackageGovernance.calls != 0 {
		t.Fatalf("did not expect governance lookup for successful resolution, got %d calls", store.listPackageGovernance.calls)
	}
}

func TestReadModuleResolveInstallTargetIncludesReplacementRecommendationForYankedVersion(t *testing.T) {
	store := newReadModuleQueryStoreStub(SampleData())
	module := newReadModule(store)

	_, err := module.ResolveInstallTarget(context.Background(), "@platform.skill/pdf-skill", "1.3.0")
	if err == nil {
		t.Fatal("expected yanked version to fail resolution")
	}

	catalogErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected catalog error, got %#v", err)
	}
	if catalogErr.Code != "version_yanked" {
		t.Fatalf("unexpected error code %q", catalogErr.Code)
	}
	if got := catalogErr.Details["replacementRecommendation"]; got != "1.2.0" {
		t.Fatalf("unexpected replacement recommendation %#v", got)
	}
	if store.listPackageVersion.calls != 1 {
		t.Fatalf("expected one package-version projection call, got %d", store.listPackageVersion.calls)
	}
	if store.listPackageVersion.packageName != "@platform.skill/pdf-skill" {
		t.Fatalf("unexpected package-version lookup %q", store.listPackageVersion.packageName)
	}
	if store.listPackageGovernance.calls != 1 {
		t.Fatalf("expected one package-governance projection call, got %d", store.listPackageGovernance.calls)
	}
	if store.listPackageGovernance.packageName != "@platform.skill/pdf-skill" {
		t.Fatalf("unexpected package-governance lookup %q", store.listPackageGovernance.packageName)
	}
	if store.listPackageGovernance.targetType != GovernanceTargetVersion {
		t.Fatalf("unexpected governance target type %q", store.listPackageGovernance.targetType)
	}
	if store.listPackageGovernance.version == nil || *store.listPackageGovernance.version != "1.3.0" {
		t.Fatalf("unexpected governance version lookup %#v", store.listPackageGovernance.version)
	}
}
