package application

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	packagingdomain "skill_registry/internal/packaging/domain"
	persist "skill_registry/internal/platform/persistence"
)

type memoryWorkspaceStore struct {
	mu      sync.Mutex
	records map[string]persist.WorkspaceRecord
}

func newMemoryWorkspaceStore() *memoryWorkspaceStore {
	return &memoryWorkspaceStore{records: map[string]persist.WorkspaceRecord{}}
}

func (s *memoryWorkspaceStore) SaveWorkspace(ctx context.Context, workspaceID string, workspacePath string, createdAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[workspaceID] = persist.WorkspaceRecord{
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	return nil
}

func (s *memoryWorkspaceStore) LoadWorkspace(ctx context.Context, workspaceID string) (persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return persist.WorkspaceRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[workspaceID]
	if !ok || strings.TrimSpace(record.WorkspacePath) == "" {
		return persist.WorkspaceRecord{}, persist.ErrWorkspaceRecordNotFound
	}
	return record, nil
}

func (s *memoryWorkspaceStore) SaveBundle(ctx context.Context, workspaceID string, bundlePath string, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *memoryWorkspaceStore) SaveValidationCache(ctx context.Context, workspaceID string, cache persist.WorkspaceValidationCache, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *memoryWorkspaceStore) SavePackageMetadata(ctx context.Context, workspaceID string, packageMetadataJSON []byte, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *memoryWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", persist.ErrWorkspaceBundleNotFound
}

func (s *memoryWorkspaceStore) ListExpired(ctx context.Context, cutoff time.Time) ([]persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *memoryWorkspaceStore) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	return ctx.Err()
}

func (s *memoryWorkspaceStore) DeleteExpired(ctx context.Context, cutoff time.Time) error {
	return ctx.Err()
}

func TestWorkspaceServicePreparePersistsManagedWorkspaceAndReturnsHandle(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewWorkspaceService(root, store)

	response, err := service.Prepare(context.Background(), packagingdomain.PrepareRequest{
		Source: packagingdomain.Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   packagingdomain.Target{PackageName: "@platform.skill/internal-runtime", Version: "1.0.0"},
		Metadata: packagingdomain.Metadata{DisplayName: "internal-runtime", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}
	if response.WorkspacePath == "" {
		t.Fatal("expected workspace handle")
	}

	record, err := store.LoadWorkspace(context.Background(), response.WorkspacePath)
	if err != nil {
		t.Fatalf("expected persisted workspace row: %v", err)
	}
	expectedManagedWorkspace := filepath.Join(root, response.WorkspacePath, "workspace")
	if record.WorkspacePath != expectedManagedWorkspace {
		t.Fatalf("expected managed workspace path %q, got %q", expectedManagedWorkspace, record.WorkspacePath)
	}
}
