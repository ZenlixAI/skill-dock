package catalog

import (
	"context"
	"sort"
	"sync"
)

type MemoryAggregateStateStoreOptions struct {
	AggregateStore AggregateStore
}

type MemoryAggregateStateStore struct {
	mu             sync.RWMutex
	aggregates     map[string]AgentSkillAggregate
	aggregateStore AggregateStore
}

func NewMemoryAggregateStateStore(data SampleCatalogData, options MemoryAggregateStateStoreOptions) *MemoryAggregateStateStore {
	store := &MemoryAggregateStateStore{
		aggregates:     make(map[string]AgentSkillAggregate),
		aggregateStore: options.AggregateStore,
	}
	if store.aggregateStore != nil {
		aggregates, err := store.aggregateStore.LoadAggregates(context.Background())
		switch {
		case err == nil:
			store.loadAggregates(aggregates)
			return store
		case err == ErrAggregateSnapshotNotFound:
		default:
			panic(err)
		}
	}
	store.loadAggregates(aggregatesFromSampleData(data))
	return store
}

func (s *MemoryAggregateStateStore) IsEmpty(context.Context) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.aggregates) == 0, nil
}

func (s *MemoryAggregateStateStore) GetAggregate(_ context.Context, packageName string) (AgentSkillAggregate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return AgentSkillAggregate{}, ErrAgentSkillAggregateNotFound
	}
	return cloneAggregate(aggregate), nil
}

func (s *MemoryAggregateStateStore) PutAggregate(ctx context.Context, aggregate AgentSkillAggregate, expectedRevision uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.aggregates[aggregate.Skill.Detail.PackageName]
	switch {
	case ok && current.Revision != expectedRevision:
		return ErrAggregateRevisionConflict
	case !ok && expectedRevision != 0:
		return ErrAggregateRevisionConflict
	}

	updated := cloneAggregate(aggregate)
	updated.Revision = expectedRevision + 1
	s.aggregates[updated.Skill.Detail.PackageName] = updated
	return s.persistLocked(ctx)
}

func (s *MemoryAggregateStateStore) ListSkills(_ context.Context, query SkillQuery) (SkillsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return SkillsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
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
		return SkillsPage{}, NewError(400, "invalid_request", "sort is invalid.", map[string]any{"sort": sortKey})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]SkillSummary, 0, len(s.aggregates))
	for _, aggregate := range s.aggregates {
		skill := aggregate.Skill.Detail
		if !query.IncludeRetired && skill.Status == SkillStatusRetired {
			continue
		}
		if query.PackageName != "" && skill.PackageName != query.PackageName {
			continue
		}
		if query.SourceType != "" && string(skill.SourceType) != query.SourceType {
			continue
		}
		if query.Query != "" && !matchesSkillQuery(skill, query.Query) {
			continue
		}
		items = append(items, skill.SkillSummary)
	}
	sortSkills(items, sortKey, query.Query)
	paged, pageInfo, err := paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return SkillsPage{}, err
	}
	return SkillsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *MemoryAggregateStateStore) ListCuratedSkills(_ context.Context, pageSize int, cursor int) (CuratedSkillsPage, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		return CuratedSkillsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": pageSize})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]CuratedSkillSummary, 0, len(s.aggregates))
	for _, aggregate := range s.aggregates {
		if aggregate.Skill.CurationReason == "" || aggregate.Skill.Detail.Status == SkillStatusRetired {
			continue
		}
		items = append(items, CuratedSkillSummary{
			SkillSummary:   aggregate.Skill.Detail.SkillSummary,
			CurationReason: aggregate.Skill.CurationReason,
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	paged, pageInfo, err := paginate(items, cursor, pageSize)
	if err != nil {
		return CuratedSkillsPage{}, err
	}
	return CuratedSkillsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *MemoryAggregateStateStore) GetSkill(_ context.Context, packageName string) (SkillDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return SkillDetail{}, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return aggregate.Skill.Detail, nil
}

func (s *MemoryAggregateStateStore) ListVersions(_ context.Context, query VersionQuery) (VersionsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return VersionsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[query.PackageName]
	if !ok {
		return VersionsPage{}, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": query.PackageName})
	}

	items := make([]SkillVersionSummary, 0, len(aggregate.Skill.Versions))
	for _, version := range aggregate.Skill.Versions {
		if !query.IncludeYanked && version.VersionStatus == VersionStatusYanked {
			continue
		}
		if !query.IncludeDeprecated && version.VersionStatus == VersionStatusDeprecated {
			continue
		}
		items = append(items, version.SkillVersionSummary)
	}
	sort.Slice(items, func(i int, j int) bool {
		return compareExactVersions(items[i].Version, items[j].Version) > 0
	})
	paged, pageInfo, err := paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return VersionsPage{}, err
	}
	return VersionsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *MemoryAggregateStateStore) GetVersion(_ context.Context, packageName string, version string) (SkillVersionDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return SkillVersionDetail{}, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	for _, item := range aggregate.Skill.Versions {
		if item.Version == version {
			return item, nil
		}
	}
	return SkillVersionDetail{}, NewError(404, "version_not_found", "Version not found.", map[string]any{
		"packageName": packageName,
		"version":     version,
	})
}

func (s *MemoryAggregateStateStore) ListPackageVersions(_ context.Context, packageName string) ([]SkillVersionDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return nil, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return append([]SkillVersionDetail(nil), aggregate.Skill.Versions...), nil
}

func (s *MemoryAggregateStateStore) ListGovernanceChanges(_ context.Context, query GovernanceQuery) (GovernancePage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return GovernancePage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []GovernanceRecord
	for _, aggregate := range s.aggregates {
		for _, record := range aggregate.Governance {
			if query.PackageName != "" && record.PackageName != query.PackageName {
				continue
			}
			if query.TargetType != "" && string(record.TargetType) != query.TargetType {
				continue
			}
			if query.Version != "" {
				if record.Version == nil || *record.Version != query.Version {
					continue
				}
			}
			items = append(items, record)
		}
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].ChangedAt.After(items[j].ChangedAt)
	})
	paged, pageInfo, err := paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return GovernancePage{}, err
	}
	return GovernancePage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *MemoryAggregateStateStore) ListPackageGovernance(_ context.Context, packageName string, targetType GovernanceTargetType, version *string) ([]GovernanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return nil, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}

	items := make([]GovernanceRecord, 0, len(aggregate.Governance))
	for _, record := range aggregate.Governance {
		if record.TargetType != targetType {
			continue
		}
		switch {
		case version == nil && record.Version != nil:
			continue
		case version != nil:
			if record.Version == nil || *record.Version != *version {
				continue
			}
		}
		items = append(items, record)
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].ChangedAt.After(items[j].ChangedAt)
	})
	return items, nil
}

func (s *MemoryAggregateStateStore) Ping(context.Context) error {
	return nil
}

func (s *MemoryAggregateStateStore) loadAggregates(aggregates []AgentSkillAggregate) {
	s.aggregates = make(map[string]AgentSkillAggregate, len(aggregates))
	for _, aggregate := range aggregates {
		s.aggregates[aggregate.Skill.Detail.PackageName] = cloneAggregate(aggregate)
	}
}

func (s *MemoryAggregateStateStore) persistLocked(ctx context.Context) error {
	if s.aggregateStore == nil {
		return nil
	}
	aggregates := make([]AgentSkillAggregate, 0, len(s.aggregates))
	for _, aggregate := range s.aggregates {
		aggregates = append(aggregates, cloneAggregate(aggregate))
	}
	if err := s.aggregateStore.SaveAggregates(ctx, aggregates); err != nil {
		return NewError(500, "internal_error", "Failed to persist catalog state.", nil)
	}
	return nil
}

func cloneAggregate(aggregate AgentSkillAggregate) AgentSkillAggregate {
	return AgentSkillAggregate{
		Revision: aggregate.Revision,
		Skill: SkillSnapshot{
			Detail:         aggregate.Skill.Detail,
			CurationReason: aggregate.Skill.CurationReason,
			Versions:       append([]SkillVersionDetail(nil), aggregate.Skill.Versions...),
		},
		Governance: append([]GovernanceRecord(nil), aggregate.Governance...),
	}
}
