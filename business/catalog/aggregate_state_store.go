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
