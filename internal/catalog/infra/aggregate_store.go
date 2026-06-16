package infra

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"

	catalogdomain "skill_registry/internal/catalog/domain"
)

type MemoryAggregateStateStoreOptions struct {
	AggregateStore catalogdomain.AggregateStore
}

type MemoryAggregateStateStore struct {
	mu             sync.RWMutex
	aggregates     map[string]catalogdomain.AgentSkillAggregate
	aggregateStore catalogdomain.AggregateStore
}

type JSONAggregateStore struct {
	path string
}

func NewJSONAggregateStore(path string) *JSONAggregateStore {
	return &JSONAggregateStore{path: path}
}

func (s *JSONAggregateStore) LoadAggregates(_ context.Context) ([]catalogdomain.AgentSkillAggregate, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, catalogdomain.ErrAggregateSnapshotNotFound
		}
		return nil, err
	}

	var aggregates []catalogdomain.AgentSkillAggregate
	if err := json.Unmarshal(content, &aggregates); err != nil {
		return nil, err
	}
	return aggregates, nil
}

func (s *JSONAggregateStore) SaveAggregates(_ context.Context, aggregates []catalogdomain.AgentSkillAggregate) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(aggregates, "", "  ")
	if err != nil {
		return err
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, content, 0o644); err != nil {
		return err
	}
	return os.Rename(tempPath, s.path)
}

func NewMemoryAggregateStateStore(data catalogdomain.SampleCatalogData, options MemoryAggregateStateStoreOptions) *MemoryAggregateStateStore {
	store := &MemoryAggregateStateStore{
		aggregates:     make(map[string]catalogdomain.AgentSkillAggregate),
		aggregateStore: options.AggregateStore,
	}
	if store.aggregateStore != nil {
		aggregates, err := store.aggregateStore.LoadAggregates(context.Background())
		switch {
		case err == nil:
			store.loadAggregates(aggregates)
			return store
		case errors.Is(err, catalogdomain.ErrAggregateSnapshotNotFound):
		default:
			panic(err)
		}
	}
	store.loadAggregates(AggregatesFromSampleData(data))
	return store
}

func (s *MemoryAggregateStateStore) IsEmpty(context.Context) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.aggregates) == 0, nil
}

func (s *MemoryAggregateStateStore) GetAggregate(_ context.Context, packageName string) (catalogdomain.AgentSkillAggregate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return catalogdomain.AgentSkillAggregate{}, catalogdomain.ErrAgentSkillAggregateNotFound
	}
	return cloneAggregateValue(aggregate), nil
}

func (s *MemoryAggregateStateStore) PutAggregate(ctx context.Context, aggregate catalogdomain.AgentSkillAggregate, expectedRevision uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.aggregates[aggregate.Skill.Detail.PackageName]
	switch {
	case ok && current.Revision != expectedRevision:
		return catalogdomain.ErrAggregateRevisionConflict
	case !ok && expectedRevision != 0:
		return catalogdomain.ErrAggregateRevisionConflict
	}

	updated := cloneAggregateValue(aggregate)
	updated.Revision = expectedRevision + 1
	s.aggregates[updated.Skill.Detail.PackageName] = updated
	return s.persistLocked(ctx)
}

func (s *MemoryAggregateStateStore) ListSkills(_ context.Context, query catalogdomain.SkillQuery) (catalogdomain.SkillsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return catalogdomain.SkillsPage{}, catalogdomain.NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}

	sortKey := query.Sort
	if sortKey == "" {
		if query.Query != "" {
			sortKey = "relevance"
		} else {
			sortKey = "updated_at_desc"
		}
	}
	if sortKey != "relevance" && sortKey != "updated_at_desc" && sortKey != "package_name_asc" {
		return catalogdomain.SkillsPage{}, catalogdomain.NewError(400, "invalid_request", "sort is invalid.", map[string]any{"sort": sortKey})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]catalogdomain.SkillSummary, 0, len(s.aggregates))
	for _, aggregate := range s.aggregates {
		skill := aggregate.Skill.Detail
		if !query.IncludeRetired && skill.Status == catalogdomain.SkillStatusRetired {
			continue
		}
		if query.PackageName != "" && skill.PackageName != query.PackageName {
			continue
		}
		if query.SourceType != "" && string(skill.SourceType) != query.SourceType {
			continue
		}
		if !matchesAllowedPackagePatterns(query.AllowedPackages, skill.PackageName) {
			continue
		}
		if query.Query != "" && !MatchesSkillQuery(skill, query.Query) {
			continue
		}
		items = append(items, skill.SkillSummary)
	}
	SortSkills(items, sortKey, query.Query)
	paged, pageInfo, err := Paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return catalogdomain.SkillsPage{}, err
	}
	return catalogdomain.SkillsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *MemoryAggregateStateStore) ListCuratedSkills(_ context.Context, query catalogdomain.CuratedSkillQuery) (catalogdomain.CuratedSkillsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return catalogdomain.CuratedSkillsPage{}, catalogdomain.NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]catalogdomain.CuratedSkillSummary, 0, len(s.aggregates))
	for _, aggregate := range s.aggregates {
		if aggregate.Skill.CurationReason == "" || aggregate.Skill.Detail.Status == catalogdomain.SkillStatusRetired {
			continue
		}
		if !matchesAllowedPackagePatterns(query.AllowedPackages, aggregate.Skill.Detail.PackageName) {
			continue
		}
		items = append(items, catalogdomain.CuratedSkillSummary{
			SkillSummary:   aggregate.Skill.Detail.SkillSummary,
			CurationReason: aggregate.Skill.CurationReason,
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	paged, pageInfo, err := Paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return catalogdomain.CuratedSkillsPage{}, err
	}
	return catalogdomain.CuratedSkillsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *MemoryAggregateStateStore) GetSkill(_ context.Context, packageName string) (catalogdomain.SkillDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return catalogdomain.SkillDetail{}, catalogdomain.NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return aggregate.Skill.Detail, nil
}

func (s *MemoryAggregateStateStore) ListVersions(_ context.Context, query catalogdomain.VersionQuery) (catalogdomain.VersionsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return catalogdomain.VersionsPage{}, catalogdomain.NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[query.PackageName]
	if !ok {
		return catalogdomain.VersionsPage{}, catalogdomain.NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": query.PackageName})
	}

	items := make([]catalogdomain.SkillVersionSummary, 0, len(aggregate.Skill.Versions))
	for _, version := range aggregate.Skill.Versions {
		if !query.IncludeYanked && version.VersionStatus == catalogdomain.VersionStatusYanked {
			continue
		}
		if !query.IncludeDeprecated && version.VersionStatus == catalogdomain.VersionStatusDeprecated {
			continue
		}
		items = append(items, version.SkillVersionSummary)
	}
	sort.Slice(items, func(i int, j int) bool {
		if !items[i].PublishedAt.Equal(items[j].PublishedAt) {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		}
		return CompareExactVersions(items[i].Version, items[j].Version) > 0
	})
	paged, pageInfo, err := Paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return catalogdomain.VersionsPage{}, err
	}
	return catalogdomain.VersionsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *MemoryAggregateStateStore) GetVersion(_ context.Context, packageName string, version string) (catalogdomain.SkillVersionDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return catalogdomain.SkillVersionDetail{}, catalogdomain.NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	for _, item := range aggregate.Skill.Versions {
		if item.Version == version {
			return item, nil
		}
	}
	return catalogdomain.SkillVersionDetail{}, catalogdomain.NewError(404, "version_not_found", "Version not found.", map[string]any{
		"packageName": packageName,
		"version":     version,
	})
}

func (s *MemoryAggregateStateStore) GetLatestResolvableVersion(_ context.Context, packageName string) (catalogdomain.SkillVersionDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return catalogdomain.SkillVersionDetail{}, catalogdomain.NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}

	var target *catalogdomain.SkillVersionDetail
	for _, item := range aggregate.Skill.Versions {
		if item.VersionStatus != catalogdomain.VersionStatusPublished || item.InstallPolicy != catalogdomain.InstallPolicyResolvable {
			continue
		}
		if target == nil || CompareExactVersions(item.Version, target.Version) > 0 {
			candidate := item
			target = &candidate
		}
	}
	if target == nil {
		return catalogdomain.SkillVersionDetail{}, catalogdomain.ErrNoResolvableVersion
	}
	return *target, nil
}

func (s *MemoryAggregateStateStore) ListPackageVersions(_ context.Context, packageName string) ([]catalogdomain.SkillVersionDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return nil, catalogdomain.NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return append([]catalogdomain.SkillVersionDetail(nil), aggregate.Skill.Versions...), nil
}

func (s *MemoryAggregateStateStore) ListPackageGovernance(_ context.Context, packageName string, targetType catalogdomain.GovernanceTargetType, version *string) ([]catalogdomain.GovernanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return nil, catalogdomain.NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	result := make([]catalogdomain.GovernanceRecord, 0, len(aggregate.Governance))
	for _, record := range aggregate.Governance {
		if record.TargetType != targetType {
			continue
		}
		if version != nil {
			if record.Version == nil || *record.Version != *version {
				continue
			}
		}
		result = append(result, record)
	}
	return result, nil
}

func (s *MemoryAggregateStateStore) ListGovernanceChanges(_ context.Context, query catalogdomain.GovernanceQuery) (catalogdomain.GovernancePage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return catalogdomain.GovernancePage{}, catalogdomain.NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[query.PackageName]
	if query.PackageName != "" && !ok {
		return catalogdomain.GovernancePage{}, catalogdomain.NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": query.PackageName})
	}

	items := make([]catalogdomain.GovernanceRecord, 0)
	if query.PackageName == "" {
		for _, aggregate := range s.aggregates {
			items = append(items, aggregate.Governance...)
		}
	} else {
		items = append(items, aggregate.Governance...)
	}
	filtered := items[:0]
	for _, record := range items {
		if query.TargetType != "" && string(record.TargetType) != query.TargetType {
			continue
		}
		if query.Version != "" {
			if record.Version == nil || *record.Version != query.Version {
				continue
			}
		}
		filtered = append(filtered, record)
	}
	sort.Slice(filtered, func(i int, j int) bool {
		return filtered[i].ChangedAt.After(filtered[j].ChangedAt)
	})
	paged, pageInfo, err := Paginate(filtered, query.Cursor, query.PageSize)
	if err != nil {
		return catalogdomain.GovernancePage{}, err
	}
	return catalogdomain.GovernancePage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *MemoryAggregateStateStore) Ping(context.Context) error {
	return nil
}

func (s *MemoryAggregateStateStore) loadAggregates(aggregates []catalogdomain.AgentSkillAggregate) {
	for _, aggregate := range aggregates {
		s.aggregates[aggregate.Skill.Detail.PackageName] = cloneAggregateValue(aggregate)
	}
}

func (s *MemoryAggregateStateStore) persistLocked(ctx context.Context) error {
	if s.aggregateStore == nil {
		return nil
	}

	aggregates := make([]catalogdomain.AgentSkillAggregate, 0, len(s.aggregates))
	for _, aggregate := range s.aggregates {
		aggregates = append(aggregates, cloneAggregateValue(aggregate))
	}
	sort.Slice(aggregates, func(i int, j int) bool {
		return aggregates[i].Skill.Detail.PackageName < aggregates[j].Skill.Detail.PackageName
	})
	return s.aggregateStore.SaveAggregates(ctx, aggregates)
}

func cloneAggregateValue(aggregate catalogdomain.AgentSkillAggregate) catalogdomain.AgentSkillAggregate {
	return catalogdomain.AgentSkillAggregate{
		Revision: aggregate.Revision,
		Skill: catalogdomain.SkillSnapshot{
			Detail:         aggregate.Skill.Detail,
			CurationReason: aggregate.Skill.CurationReason,
			Versions:       append([]catalogdomain.SkillVersionDetail(nil), aggregate.Skill.Versions...),
		},
		Governance: append([]catalogdomain.GovernanceRecord(nil), aggregate.Governance...),
	}
}
