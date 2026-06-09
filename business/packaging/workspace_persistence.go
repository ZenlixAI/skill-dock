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

type packagingWorkspaceBacking struct {
	workspaceRoot   string
	workspaceSource workspaceSourceModule
	workspaceStore  persist.WorkspaceStore
	workspaceLocker *workspaceLocker
}

func newPackagingWorkspaceBacking(tempRoot string, store persist.WorkspaceStore) *packagingWorkspaceBacking {
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	return &packagingWorkspaceBacking{
		workspaceRoot:   tempRoot,
		workspaceSource: newWorkspaceSourceModule(tempRoot, 24*time.Hour),
		workspaceStore:  store,
		workspaceLocker: newWorkspaceLocker(),
	}
}

func (w *packagingWorkspaceBacking) lockWorkspace(ctx context.Context, workspaceID string) (context.Context, func(), error) {
	return w.workspaceLocker.lock(ctx, workspaceID, w.workspaceStore != nil)
}

func (w *packagingWorkspaceBacking) persistManagedWorkspace(ctx context.Context, workspacePath string) (string, string, error) {
	if err := packagingContextError(ctx); err != nil {
		return "", "", err
	}

	workspaceID := workspaceHandle()
	managedWorkspace := w.managedWorkspacePath(workspaceID)
	managedRoot, err := w.managedWorkspaceRootFromPath(workspaceID, managedWorkspace)
	if err != nil {
		return "", "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	if err := os.MkdirAll(managedRoot, 0o755); err != nil {
		return "", "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	if err := os.Rename(workspacePath, managedWorkspace); err != nil {
		return "", "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	if err := w.workspaceStore.SaveWorkspace(ctx, workspaceID, managedWorkspace, time.Now().UTC()); err != nil {
		_ = os.RemoveAll(managedRoot)
		return "", "", catalog.NewError(500, "internal_error", "Failed to persist packaging workspace.", nil)
	}
	return workspaceID, managedWorkspace, nil
}

func (w *packagingWorkspaceBacking) materializeWorkspace(ctx context.Context, workspacePath string) (string, func(), error) {
	if err := packagingContextError(ctx); err != nil {
		return "", nil, err
	}
	if w.workspaceStore == nil {
		return workspacePath, func() {}, nil
	}

	record, err := w.loadManagedWorkspaceRecord(ctx, workspacePath)
	if err != nil {
		return "", nil, err
	}
	return record.WorkspacePath, func() {}, nil
}

func (w *packagingWorkspaceBacking) loadManagedWorkspaceRecord(ctx context.Context, workspacePath string) (persist.WorkspaceRecord, error) {
	record, err := w.workspaceStore.LoadWorkspace(ctx, workspacePath)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return persist.WorkspaceRecord{}, catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
		}
		return persist.WorkspaceRecord{}, catalog.NewError(500, "internal_error", "Failed to load packaging workspace.", nil)
	}
	if _, err := w.managedWorkspaceRootFromPath(workspacePath, record.WorkspacePath); err != nil {
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

func (w *packagingWorkspaceBacking) managedWorkspacePath(workspaceID string) string {
	return filepath.Join(w.workspaceRoot, workspaceID, "workspace")
}

func (w *packagingWorkspaceBacking) managedBundlePath(workspaceID string, target Target) string {
	return filepath.Join(w.workspaceRoot, workspaceID, ".dist", packageBasename(target.PackageName)+"-"+target.Version+".tgz")
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

func (w *packagingWorkspaceBacking) managedWorkspaceRoot(workspaceID string) string {
	return filepath.Join(w.workspaceRoot, workspaceID)
}

func (w *packagingWorkspaceBacking) managedWorkspaceRootFromPath(workspaceID string, workspacePath string) (string, error) {
	expectedWorkspace := filepath.Clean(w.managedWorkspacePath(workspaceID))
	actualWorkspace := filepath.Clean(workspacePath)
	if actualWorkspace != expectedWorkspace {
		return "", errors.New("managed workspace path must match canonical layout")
	}
	managedRoot := w.managedWorkspaceRoot(workspaceID)
	if err := ensureManagedPathWithinRoot(w.workspaceRoot, managedRoot); err != nil {
		return "", err
	}
	return managedRoot, nil
}

func (w *packagingWorkspaceBacking) validateManagedBundlePath(workspaceID string, target Target, bundlePath string) error {
	expectedBundle := filepath.Clean(w.managedBundlePath(workspaceID, target))
	actualBundle := filepath.Clean(bundlePath)
	if actualBundle != expectedBundle {
		return errors.New("managed bundle path must match canonical layout")
	}
	return ensureManagedPathWithinRoot(w.workspaceRoot, actualBundle)
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
