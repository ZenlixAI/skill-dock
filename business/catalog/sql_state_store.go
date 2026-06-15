package catalog

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	internaldomain "skill_registry/internal/catalog/domain"
	internalinfra "skill_registry/internal/catalog/infra"
)

type SQLStateStore struct {
	inner *internalinfra.SQLStateStore
}

func NewSQLStateStore(db *sql.DB, dialect string) *SQLStateStore {
	return &SQLStateStore{inner: internalinfra.NewSQLStateStore(db, dialect)}
}

func (s *SQLStateStore) Migrate(ctx context.Context) error {
	return s.inner.Migrate(ctx)
}

func (s *SQLStateStore) IsEmpty(ctx context.Context) (bool, error) {
	return s.inner.IsEmpty(ctx)
}

func (s *SQLStateStore) GetAggregate(ctx context.Context, packageName string) (AgentSkillAggregate, error) {
	aggregate, err := s.inner.GetAggregate(ctx, packageName)
	if err != nil {
		if errors.Is(err, internaldomain.ErrAgentSkillAggregateNotFound) {
			return AgentSkillAggregate{}, ErrAgentSkillAggregateNotFound
		}
		return AgentSkillAggregate{}, err
	}
	return fromInternalAggregate(aggregate), nil
}

func (s *SQLStateStore) PutAggregate(ctx context.Context, aggregate AgentSkillAggregate, expectedRevision uint64) error {
	if err := s.inner.PutAggregate(ctx, toInternalAggregate(aggregate), expectedRevision); err != nil {
		if errors.Is(err, internaldomain.ErrAggregateRevisionConflict) {
			return ErrAggregateRevisionConflict
		}
		return err
	}
	return nil
}

func (s *SQLStateStore) ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error) {
	return s.inner.ListSkills(ctx, query)
}

func (s *SQLStateStore) ListCuratedSkills(ctx context.Context, query CuratedSkillQuery) (CuratedSkillsPage, error) {
	return s.inner.ListCuratedSkills(ctx, query)
}

func (s *SQLStateStore) GetSkill(ctx context.Context, packageName string) (SkillDetail, error) {
	return s.inner.GetSkill(ctx, packageName)
}

func (s *SQLStateStore) ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error) {
	return s.inner.ListVersions(ctx, query)
}

func (s *SQLStateStore) GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error) {
	return s.inner.GetVersion(ctx, packageName, version)
}

func (s *SQLStateStore) GetLatestResolvableVersion(ctx context.Context, packageName string) (SkillVersionDetail, error) {
	return s.inner.GetLatestResolvableVersion(ctx, packageName)
}

func (s *SQLStateStore) ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error) {
	return s.inner.ListGovernanceChanges(ctx, query)
}

func (s *SQLStateStore) ListPackageVersions(ctx context.Context, packageName string) ([]SkillVersionDetail, error) {
	return s.inner.ListPackageVersions(ctx, packageName)
}

func (s *SQLStateStore) ListPackageGovernance(ctx context.Context, packageName string, targetType GovernanceTargetType, version *string) ([]GovernanceRecord, error) {
	return s.inner.ListPackageGovernance(ctx, packageName, targetType, version)
}

func (s *SQLStateStore) Ping(ctx context.Context) error {
	return s.inner.Ping(ctx)
}

func bindSQL(dialect string, query string) string {
	return internalinfra.BindSQL(dialect, query)
}

func matchesAllowedPackagePatterns(patterns []string, packageName string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "*") {
			if strings.HasPrefix(packageName, strings.TrimSuffix(pattern, "*")) {
				return true
			}
			continue
		}
		if pattern == packageName {
			return true
		}
	}
	return false
}
