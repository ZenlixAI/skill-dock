package packaging

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

func (s *Service) persistManagedWorkspace(ctx context.Context, workspacePath string) (string, string, error) {
	if err := packagingContextError(ctx); err != nil {
		return "", "", err
	}

	workspaceID := workspaceHandle()
	managedWorkspace := s.managedWorkspacePath(workspaceID)
	managedRoot, err := s.managedWorkspaceRootFromPath(workspaceID, managedWorkspace)
	if err != nil {
		return "", "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	if err := os.MkdirAll(managedRoot, 0o755); err != nil {
		return "", "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	if err := os.Rename(workspacePath, managedWorkspace); err != nil {
		return "", "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	if err := s.workspaceStore.SaveWorkspace(ctx, workspaceID, managedWorkspace, time.Now().UTC()); err != nil {
		_ = os.RemoveAll(managedRoot)
		return "", "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	return workspaceID, managedWorkspace, nil
}

func (s *Service) materializeWorkspace(ctx context.Context, workspacePath string) (string, func(), error) {
	if err := packagingContextError(ctx); err != nil {
		return "", nil, err
	}
	if s.workspaceStore == nil {
		return workspacePath, func() {}, nil
	}

	record, err := s.loadManagedWorkspaceRecord(ctx, workspacePath)
	if err != nil {
		return "", nil, err
	}
	return record.WorkspacePath, func() {}, nil
}

func (s *Service) loadManagedWorkspaceRecord(ctx context.Context, workspacePath string) (persist.WorkspaceRecord, error) {
	record, err := s.workspaceStore.LoadWorkspace(ctx, workspacePath)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return persist.WorkspaceRecord{}, catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
		}
		return persist.WorkspaceRecord{}, catalog.NewError(500, "internal_error", "Failed to load packaging workspace.", nil)
	}
	if _, err := s.managedWorkspaceRootFromPath(workspacePath, record.WorkspacePath); err != nil {
		return persist.WorkspaceRecord{}, catalog.NewError(500, "internal_error", "Failed to load packaging workspace.", nil)
	}
	info, err := os.Stat(record.WorkspacePath)
	if err != nil || !info.IsDir() {
		if errors.Is(err, os.ErrNotExist) || err == nil {
			return persist.WorkspaceRecord{}, catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
		}
		return persist.WorkspaceRecord{}, catalog.NewError(500, "internal_error", "Failed to load packaging workspace.", nil)
	}

	return record, nil
}

func (s *Service) managedWorkspacePath(workspaceID string) string {
	return filepath.Join(s.workspaceRoot, workspaceID, "workspace")
}

func (s *Service) managedBundlePath(workspaceID string, target Target) string {
	return filepath.Join(s.workspaceRoot, workspaceID, ".dist", packageBasename(target.PackageName)+"-"+target.Version+".tgz")
}

func ensureManagedPathWithinRoot(root string, path string) error {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(path) == "" {
		return errors.New("managed path is required")
	}
	relativePath, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return err
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return errors.New("managed path must stay within root")
	}
	return nil
}

func (s *Service) managedWorkspaceRoot(workspaceID string) string {
	return filepath.Join(s.workspaceRoot, workspaceID)
}

func (s *Service) managedWorkspaceRootFromPath(workspaceID string, workspacePath string) (string, error) {
	expectedWorkspace := filepath.Clean(s.managedWorkspacePath(workspaceID))
	actualWorkspace := filepath.Clean(workspacePath)
	if actualWorkspace != expectedWorkspace {
		return "", errors.New("managed workspace path must match canonical layout")
	}
	managedRoot := s.managedWorkspaceRoot(workspaceID)
	if err := ensureManagedPathWithinRoot(s.workspaceRoot, managedRoot); err != nil {
		return "", err
	}
	return managedRoot, nil
}

func (s *Service) validateManagedBundlePath(workspaceID string, target Target, bundlePath string) error {
	expectedBundle := filepath.Clean(s.managedBundlePath(workspaceID, target))
	actualBundle := filepath.Clean(bundlePath)
	if actualBundle != expectedBundle {
		return errors.New("managed bundle path must match canonical layout")
	}
	return ensureManagedPathWithinRoot(s.workspaceRoot, actualBundle)
}

func workspaceHandle() string {
	dir, _ := os.MkdirTemp("", "ws-")
	if dir == "" {
		return "workspace-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	_ = os.RemoveAll(dir)
	return filepath.Base(dir)
}

func workspaceBundleHandle(workspacePath string, target Target) string {
	return "workspace://" + workspacePath + "/" + packageBasename(target.PackageName) + "-" + target.Version + ".tgz"
}
