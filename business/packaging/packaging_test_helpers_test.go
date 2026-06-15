package packaging

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"skill_registry/business/catalog"
	artifactadapter "skill_registry/internal/platform/artifact"
	persist "skill_registry/internal/platform/persistence"
)

type stubReleaseRecordOutcome struct {
	result catalog.ReleaseRecordResult
	err    error
}

type stubReleaseRecordCreator struct {
	calls   int
	results []stubReleaseRecordOutcome
}

func (s *stubReleaseRecordCreator) CreateReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error) {
	s.calls++
	index := s.calls - 1
	if index >= len(s.results) {
		return catalog.ReleaseRecordResult{}, catalog.NewError(500, "internal_error", "unexpected extra release record call", nil)
	}
	return s.results[index].result, s.results[index].err
}

type capturePackagePublisher struct {
	calls     int
	lastInput artifactadapter.NPMPublishInput
	err       error
	started   chan struct{}
	block     chan struct{}
	once      sync.Once
}

func (s *capturePackagePublisher) PublishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) error {
	s.calls++
	s.lastInput = input
	if s.started != nil {
		s.once.Do(func() {
			close(s.started)
		})
	}
	if s.block != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.block:
		}
	}
	return s.err
}

type cancelAwareWorkspaceStore struct{}

func (cancelAwareWorkspaceStore) SaveWorkspace(ctx context.Context, workspaceID string, workspacePath string, createdAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (cancelAwareWorkspaceStore) LoadWorkspace(ctx context.Context, workspaceID string) (persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return persist.WorkspaceRecord{}, err
	}
	return persist.WorkspaceRecord{}, persist.ErrWorkspaceRecordNotFound
}

func (cancelAwareWorkspaceStore) SaveBundle(ctx context.Context, workspaceID string, bundlePath string, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (cancelAwareWorkspaceStore) SaveValidationCache(ctx context.Context, workspaceID string, cache persist.WorkspaceValidationCache, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (cancelAwareWorkspaceStore) SavePackageMetadata(ctx context.Context, workspaceID string, packageMetadataJSON []byte, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (cancelAwareWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", persist.ErrWorkspaceBundleNotFound
}

func (cancelAwareWorkspaceStore) ListExpired(ctx context.Context, cutoff time.Time) ([]persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (cancelAwareWorkspaceStore) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	return ctx.Err()
}

func (cancelAwareWorkspaceStore) DeleteExpired(ctx context.Context, cutoff time.Time) error {
	return ctx.Err()
}

type memoryWorkspaceStore struct {
	mu               sync.Mutex
	afterListExpired func()
	records          map[string]persist.WorkspaceRecord
}

type countingWorkspaceStore struct {
	*memoryWorkspaceStore
	loadWorkspaceCalls int
	loadBundleCalls    int
}

func newMemoryWorkspaceStore() *memoryWorkspaceStore {
	return &memoryWorkspaceStore{records: map[string]persist.WorkspaceRecord{}}
}

func newCountingWorkspaceStore() *countingWorkspaceStore {
	return &countingWorkspaceStore{memoryWorkspaceStore: newMemoryWorkspaceStore()}
}

func (s *memoryWorkspaceStore) SaveWorkspace(ctx context.Context, workspaceID string, workspacePath string, createdAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	record := persist.WorkspaceRecord{
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	if existing, ok := s.records[workspaceID]; ok {
		record.CreatedAt = existing.CreatedAt
	}
	s.records[workspaceID] = record
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
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[workspaceID]
	if !ok || strings.TrimSpace(record.WorkspacePath) == "" {
		return persist.ErrWorkspaceRecordNotFound
	}
	record.BundlePath = bundlePath
	record.UpdatedAt = updatedAt
	s.records[workspaceID] = record
	return nil
}

func (s *memoryWorkspaceStore) SaveValidationCache(ctx context.Context, workspaceID string, cache persist.WorkspaceValidationCache, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[workspaceID]
	if !ok || strings.TrimSpace(record.WorkspacePath) == "" {
		return persist.ErrWorkspaceRecordNotFound
	}
	cacheCopy := cache
	cacheCopy.Components = append([]string(nil), cache.Components...)
	cacheCopy.Warnings = append([]string(nil), cache.Warnings...)
	cacheCopy.Errors = append([]string(nil), cache.Errors...)
	record.ValidationCache = &cacheCopy
	record.UpdatedAt = updatedAt
	s.records[workspaceID] = record
	return nil
}

func (s *memoryWorkspaceStore) SavePackageMetadata(ctx context.Context, workspaceID string, packageMetadataJSON []byte, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[workspaceID]
	if !ok || strings.TrimSpace(record.WorkspacePath) == "" {
		return persist.ErrWorkspaceRecordNotFound
	}
	record.PackageMetadataJSON = append([]byte(nil), packageMetadataJSON...)
	record.UpdatedAt = updatedAt
	s.records[workspaceID] = record
	return nil
}

func (s *memoryWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[workspaceID]
	if !ok || strings.TrimSpace(record.WorkspacePath) == "" {
		return "", persist.ErrWorkspaceRecordNotFound
	}
	if strings.TrimSpace(record.BundlePath) == "" {
		return "", persist.ErrWorkspaceBundleNotFound
	}
	return record.BundlePath, nil
}

func (s *memoryWorkspaceStore) ListExpired(ctx context.Context, cutoff time.Time) ([]persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	records := []persist.WorkspaceRecord{}
	for _, record := range s.records {
		if !record.UpdatedAt.IsZero() && record.UpdatedAt.Before(cutoff) {
			records = append(records, record)
		}
	}
	hook := s.afterListExpired
	s.mu.Unlock()
	if hook != nil {
		hook()
	}
	return records, nil
}

func (s *memoryWorkspaceStore) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.records, workspaceID)
	return nil
}

func (s *memoryWorkspaceStore) DeleteExpired(ctx context.Context, cutoff time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for workspaceID, record := range s.records {
		if !record.UpdatedAt.IsZero() && record.UpdatedAt.Before(cutoff) {
			delete(s.records, workspaceID)
		}
	}
	return nil
}

func (s *countingWorkspaceStore) LoadWorkspace(ctx context.Context, workspaceID string) (persist.WorkspaceRecord, error) {
	s.loadWorkspaceCalls++
	return s.memoryWorkspaceStore.LoadWorkspace(ctx, workspaceID)
}

func (s *countingWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) (string, error) {
	s.loadBundleCalls++
	return s.memoryWorkspaceStore.LoadBundle(ctx, workspaceID)
}

func writeSkillMarkdownFixture(t *testing.T, dir string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMarkdownFixture()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func skillMarkdownFixture() string {
	return `---
name: zoom-out
description: Broader context
---

Prompt body.
`
}

func copyExternalSkillFixture(t *testing.T, fixtureName string) string {
	t.Helper()

	sourceDir := filepath.Join("testdata", "external-skills", fixtureName)
	targetDir := t.TempDir()
	if err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relativePath == "." {
			return nil
		}
		targetPath := filepath.Join(targetDir, relativePath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(targetPath, content, 0o644)
	}); err != nil {
		t.Fatalf("copy fixture %s: %v", fixtureName, err)
	}
	return targetDir
}

func writeCapabilityFixture(t *testing.T, dir string) {
	t.Helper()

	capabilityPath := filepath.Join(dir, "docs", "README.md")
	if err := os.MkdirAll(filepath.Dir(capabilityPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capabilityPath, []byte("# Capability\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func createGitSkillRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	if err := runGitCommand("", "init", repoDir); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGitCommand(repoDir, "config", "user.name", "Codex Test"); err != nil {
		t.Fatalf("git config user.name failed: %v", err)
	}
	if err := runGitCommand(repoDir, "config", "user.email", "codex@example.com"); err != nil {
		t.Fatalf("git config user.email failed: %v", err)
	}

	writeSkillMarkdownFixture(t, repoDir)

	if err := runGitCommand(repoDir, "add", "SKILL.md"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := runGitCommand(repoDir, "commit", "-m", "add skill"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
	return repoDir
}

func readNormalizedFromImportFlag(t *testing.T, workspacePath string) bool {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(workspacePath, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}

	var payload struct {
		Skill struct {
			NormalizedFromImport bool `json:"normalizedFromImport"`
		} `json:"skill"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("unmarshal package.json: %v", err)
	}
	return payload.Skill.NormalizedFromImport
}

func buildSourceArchiveBase64FromDirectory(t *testing.T, root string) string {
	t.Helper()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if shouldIgnorePackagingPath(relativePath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relativePath
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tarWriter.Write(content)
		return err
	})
	if err != nil {
		t.Fatalf("build source archive: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return base64.StdEncoding.EncodeToString(archive.Bytes())
}

func readTarGzEntries(t *testing.T, archive []byte) map[string][]byte {
	t.Helper()

	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("open gzip archive: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	entries := map[string][]byte{}
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("read tar entry: %v", err)
			break
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read tar entry content: %v", err)
		}
		entries[header.Name] = content
	}
	return entries
}

func sortedMapKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
