package application

import (
	"context"
	"testing"

	"skill_registry/internal/catalog/domain"
)

type stubQueryStore struct {
	listed bool
}

func (s *stubQueryStore) ListSkills(context.Context, domain.SkillQuery) (domain.SkillsPage, error) {
	s.listed = true
	return domain.SkillsPage{}, nil
}

func (s *stubQueryStore) ListCuratedSkills(context.Context, domain.CuratedSkillQuery) (domain.CuratedSkillsPage, error) {
	panic("unexpected call to ListCuratedSkills")
}

func (s *stubQueryStore) GetSkill(context.Context, string) (domain.SkillDetail, error) {
	panic("unexpected call to GetSkill")
}

func (s *stubQueryStore) ListVersions(context.Context, domain.VersionQuery) (domain.VersionsPage, error) {
	panic("unexpected call to ListVersions")
}

func (s *stubQueryStore) GetVersion(context.Context, string, string) (domain.SkillVersionDetail, error) {
	panic("unexpected call to GetVersion")
}

func (s *stubQueryStore) GetLatestResolvableVersion(context.Context, string) (domain.SkillVersionDetail, error) {
	panic("unexpected call to GetLatestResolvableVersion")
}

func (s *stubQueryStore) ListGovernanceChanges(context.Context, domain.GovernanceQuery) (domain.GovernancePage, error) {
	panic("unexpected call to ListGovernanceChanges")
}

func (s *stubQueryStore) ListPackageVersions(context.Context, string) ([]domain.SkillVersionDetail, error) {
	panic("unexpected call to ListPackageVersions")
}

func (s *stubQueryStore) ListPackageGovernance(context.Context, string, domain.GovernanceTargetType, *string) ([]domain.GovernanceRecord, error) {
	panic("unexpected call to ListPackageGovernance")
}

func TestReadServiceListSkillsDelegatesToQueryStore(t *testing.T) {
	store := &stubQueryStore{}
	service := NewReadService(store)

	if _, err := service.ListSkills(context.Background(), domain.SkillQuery{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !store.listed {
		t.Fatal("expected query store to be called")
	}
}
