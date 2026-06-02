package catalog

import (
	"context"
	"errors"
)

var (
	ErrAgentSkillAggregateNotFound = errors.New("agent skill aggregate not found")
	ErrAggregateRevisionConflict   = errors.New("agent skill aggregate revision conflict")
)

type AgentSkillAggregate struct {
	Revision   uint64             `json:"revision"`
	Skill      SkillSnapshot      `json:"skill"`
	Governance []GovernanceRecord `json:"governance"`
}

type CatalogStateStore interface {
	IsEmpty(ctx context.Context) (bool, error)
	GetAggregate(ctx context.Context, packageName string) (AgentSkillAggregate, error)
	PutAggregate(ctx context.Context, aggregate AgentSkillAggregate, expectedRevision uint64) error
	ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error)
	ListCuratedSkills(ctx context.Context, pageSize int, cursor int) (CuratedSkillsPage, error)
	ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error)
	ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error)
	Ping(ctx context.Context) error
}
