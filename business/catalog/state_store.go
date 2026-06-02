package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var ErrAggregateSnapshotNotFound = errors.New("aggregate snapshot not found")

type AggregateStore interface {
	LoadAggregates(ctx context.Context) ([]AgentSkillAggregate, error)
	SaveAggregates(ctx context.Context, aggregates []AgentSkillAggregate) error
}

type JSONAggregateStore struct {
	path string
}

func NewJSONAggregateStore(path string) *JSONAggregateStore {
	return &JSONAggregateStore{path: path}
}

func (s *JSONAggregateStore) LoadAggregates(_ context.Context) ([]AgentSkillAggregate, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrAggregateSnapshotNotFound
		}
		return nil, err
	}

	var aggregates []AgentSkillAggregate
	if err := json.Unmarshal(content, &aggregates); err != nil {
		return nil, err
	}
	return aggregates, nil
}

func (s *JSONAggregateStore) SaveAggregates(_ context.Context, aggregates []AgentSkillAggregate) error {
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
