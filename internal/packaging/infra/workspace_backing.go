package infra

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	catalog "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
	persist "skill_registry/internal/platform/persistence"
)

type packagingWorkspaceBacking struct {
	workspaceRoot   string
	workspaceSource workspaceSourceModule
	workspaceStore  persist.WorkspaceStore
	workspaceLocker *workspaceLocker
}

type WorkspaceBacking = packagingWorkspaceBacking

func NewWorkspaceBacking(tempRoot string, store persist.WorkspaceStore) *WorkspaceBacking {
	return newPackagingWorkspaceBacking(tempRoot, store)
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

func (w *packagingWorkspaceBacking) LockWorkspace(ctx context.Context, workspaceID string) (context.Context, func(), error) {
	return w.lockWorkspace(ctx, workspaceID)
}

func (w *packagingWorkspaceBacking) persistManagedWorkspace(ctx context.Context, workspacePath string) (string, string, error) {
	if err := packagingContextError(ctx); err != nil {
		return "", "", err
	}

	workspaceID := workspaceHandle()
	managedWorkspace := w.managedWorkspacePath(workspaceID)
	managedRoot, err := w.canonicalManagedWorkspaceRoot(workspaceID)
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

func (w *packagingWorkspaceBacking) materializeWorkspace(ctx context.Context, workspaceHandle string) (string, func(), error) {
	if err := packagingContextError(ctx); err != nil {
		return "", nil, err
	}
	if w.workspaceStore == nil {
		return workspaceHandle, func() {}, nil
	}

	record, err := w.loadManagedWorkspaceRecord(ctx, workspaceHandle)
	if err != nil {
		return "", nil, err
	}
	return record.WorkspacePath, func() {}, nil
}

func (w *packagingWorkspaceBacking) loadManagedWorkspaceRecord(ctx context.Context, workspaceHandle string) (persist.WorkspaceRecord, error) {
	record, err := w.workspaceStore.LoadWorkspace(ctx, workspaceHandle)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return persist.WorkspaceRecord{}, catalog.NewError(422, "invalid_request", "workspaceHandle must reference an existing packaging workspace.", map[string]any{"workspaceHandle": workspaceHandle})
		}
		return persist.WorkspaceRecord{}, catalog.NewError(500, "internal_error", "Failed to load packaging workspace.", nil)
	}
	managedWorkspacePath := w.managedWorkspacePath(workspaceHandle)
	if _, err := w.canonicalManagedWorkspaceRoot(workspaceHandle); err != nil {
		return persist.WorkspaceRecord{}, catalog.NewError(500, "internal_error", "Failed to load packaging workspace.", nil)
	}
	info, err := os.Stat(managedWorkspacePath)
	if err != nil || !info.IsDir() {
		if errors.Is(err, os.ErrNotExist) || err == nil {
			return persist.WorkspaceRecord{}, catalog.NewError(422, "invalid_request", "workspaceHandle must reference an existing packaging workspace.", map[string]any{"workspaceHandle": workspaceHandle})
		}
		return persist.WorkspaceRecord{}, catalog.NewError(500, "internal_error", "Failed to load packaging workspace.", nil)
	}

	record.WorkspacePath = managedWorkspacePath
	return record, nil
}

func (w *packagingWorkspaceBacking) managedWorkspacePath(workspaceID string) string {
	return filepath.Join(w.workspaceRoot, workspaceID, "workspace")
}

func (w *packagingWorkspaceBacking) managedBundlePath(workspaceID string, target packagingdomain.Target) string {
	return filepath.Join(w.workspaceRoot, workspaceID, ".dist", packagingdomain.PackageBasename(target.PackageName)+"-"+target.Version+".tgz")
}

func (w *packagingWorkspaceBacking) ManagedBundlePath(workspaceID string, target packagingdomain.Target) string {
	return w.managedBundlePath(workspaceID, target)
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

func (w *packagingWorkspaceBacking) canonicalManagedWorkspaceRoot(workspaceID string) (string, error) {
	managedRoot := w.managedWorkspaceRoot(workspaceID)
	if err := ensureManagedPathWithinRoot(w.workspaceRoot, managedRoot); err != nil {
		return "", err
	}
	return managedRoot, nil
}

func (w *packagingWorkspaceBacking) managedWorkspaceRootFromPath(workspaceID string, workspacePath string) (string, error) {
	expectedWorkspace := filepath.Clean(w.managedWorkspacePath(workspaceID))
	actualWorkspace := filepath.Clean(workspacePath)
	if actualWorkspace != expectedWorkspace {
		return "", errors.New("managed workspace path must match canonical layout")
	}
	return w.canonicalManagedWorkspaceRoot(workspaceID)
}

func (w *packagingWorkspaceBacking) validateManagedBundlePath(workspaceID string, target packagingdomain.Target, bundlePath string) error {
	expectedBundle := filepath.Clean(w.managedBundlePath(workspaceID, target))
	actualBundle := filepath.Clean(bundlePath)
	if actualBundle != expectedBundle {
		return errors.New("managed bundle path must match canonical layout")
	}
	return ensureManagedPathWithinRoot(w.workspaceRoot, actualBundle)
}

func (w *packagingWorkspaceBacking) HasWorkspaceStore() bool {
	return w.workspaceStore != nil
}

func workspaceHandle() string {
	dir, _ := os.MkdirTemp("", "ws-")
	if dir == "" {
		return "workspace-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	_ = os.RemoveAll(dir)
	return filepath.Base(dir)
}

func workspaceBundleHandle(workspaceHandle string, target packagingdomain.Target) string {
	return "workspace://" + workspaceHandle + "/" + packagingdomain.PackageBasename(target.PackageName) + "-" + target.Version + ".tgz"
}

func packagingContextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	switch err := ctx.Err(); {
	case err == nil:
		return nil
	case err == context.DeadlineExceeded:
		return catalog.NewError(504, "request_timeout", "Request context deadline exceeded during packaging.", nil)
	default:
		return catalog.NewError(499, "request_canceled", "Request context was canceled during packaging.", nil)
	}
}
