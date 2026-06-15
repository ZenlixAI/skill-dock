package packaging

import (
	"context"
	"time"

	packagingapplication "skill_registry/internal/packaging/application"
	persist "skill_registry/internal/platform/persistence"
)

type packagingWorkspaceRuntime struct {
	impl *packagingapplication.WorkspaceService
}

func newPackagingWorkspaceRuntime(tempRoot string, store persist.WorkspaceStore) *packagingWorkspaceRuntime {
	return &packagingWorkspaceRuntime{impl: packagingapplication.NewWorkspaceService(tempRoot, store)}
}

func (r *packagingWorkspaceRuntime) Prepare(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	return r.impl.Prepare(ctx, input)
}

func (r *packagingWorkspaceRuntime) Validate(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	return r.impl.Validate(ctx, input)
}

func (r *packagingWorkspaceRuntime) Pack(ctx context.Context, input PackRequest) (PackResponse, error) {
	return r.impl.Pack(ctx, input)
}

func (r *packagingWorkspaceRuntime) Cleanup(ctx context.Context, workspacePath string) error {
	return r.impl.Cleanup(ctx, workspacePath)
}

func (r *packagingWorkspaceRuntime) BundlePath(ctx context.Context, workspacePath string, target Target) (string, error) {
	return r.impl.BundlePath(ctx, workspacePath, target)
}

func (r *packagingWorkspaceRuntime) BundleFilePath(ctx context.Context, workspacePath string, target Target) (string, error) {
	return r.impl.BundleFilePath(ctx, workspacePath, target)
}

func (r *packagingWorkspaceRuntime) BundleContent(ctx context.Context, workspacePath string, target Target) ([]byte, error) {
	return r.impl.BundleContent(ctx, workspacePath, target)
}

func (r *packagingWorkspaceRuntime) PackageMetadata(ctx context.Context, workspacePath string) (map[string]any, error) {
	return r.impl.PackageMetadata(ctx, workspacePath)
}

func (r *packagingWorkspaceRuntime) PruneExpiredWorkspaces(ctx context.Context, ttl time.Duration) error {
	return r.impl.PruneExpiredWorkspaces(ctx, ttl)
}
