package application

import (
	"context"
	"time"

	packagingdomain "skill_registry/internal/packaging/domain"
	packaginginfra "skill_registry/internal/packaging/infra"
	persist "skill_registry/internal/platform/persistence"
)

type WorkspaceService struct {
	workspace   *packaginginfra.WorkspaceBacking
	adapter     packaginginfra.WorkspaceAdapter
	bundleRules packagingdomain.BundleRules
}

type publishedReleaseMaterial struct {
	packResponse packagingdomain.PackResponse
	metadata     packaginginfra.WorkspacePackageMetadata
	tarballPath  string
	cleanup      func()
}

func NewWorkspaceService(tempRoot string, store persist.WorkspaceStore) *WorkspaceService {
	return NewWorkspaceServiceWithBacking(packaginginfra.NewWorkspaceBacking(tempRoot, store))
}

func NewWorkspaceServiceWithBacking(workspace *packaginginfra.WorkspaceBacking) *WorkspaceService {
	return &WorkspaceService{
		workspace:   workspace,
		adapter:     packaginginfra.NewWorkspaceAdapter(workspace),
		bundleRules: packagingdomain.NewBundleRules(),
	}
}

func (r *WorkspaceService) Prepare(ctx context.Context, input packagingdomain.PrepareRequest) (packagingdomain.PrepareResponse, error) {
	if err := packagingdomain.ValidateTarget(input.Target); err != nil {
		return packagingdomain.PrepareResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return packagingdomain.PrepareResponse{}, err
	}

	return r.adapter.Prepare(ctx, input, r.bundleRules)
}

func (r *WorkspaceService) Validate(ctx context.Context, input packagingdomain.ValidateRequest) (packagingdomain.ValidateResponse, error) {
	if err := packagingdomain.ValidateTarget(input.Target); err != nil {
		return packagingdomain.ValidateResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return packagingdomain.ValidateResponse{}, err
	}
	ctx, unlock, err := r.workspace.LockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return packagingdomain.ValidateResponse{}, err
	}
	defer unlock()

	return r.validateLocked(ctx, input)
}

func (r *WorkspaceService) validateLocked(ctx context.Context, input packagingdomain.ValidateRequest) (packagingdomain.ValidateResponse, error) {
	return r.adapter.ValidateLocked(ctx, input, r.bundleRules)
}

func (r *WorkspaceService) Pack(ctx context.Context, input packagingdomain.PackRequest) (packagingdomain.PackResponse, error) {
	if err := packagingdomain.ValidateTarget(input.Target); err != nil {
		return packagingdomain.PackResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return packagingdomain.PackResponse{}, err
	}
	ctx, unlock, err := r.workspace.LockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return packagingdomain.PackResponse{}, err
	}
	defer unlock()

	result, err := r.adapter.PackLocked(ctx, input, r.bundleRules)
	if err != nil {
		return packagingdomain.PackResponse{}, err
	}
	defer result.Cleanup()()

	return result.Response, nil
}

func (r *WorkspaceService) Cleanup(ctx context.Context, workspacePath string) error {
	if err := packagingContextError(ctx); err != nil {
		return err
	}
	ctx, unlock, err := r.workspace.LockWorkspace(ctx, workspacePath)
	if err != nil {
		return err
	}
	defer unlock()

	return r.adapter.CleanupLocked(ctx, workspacePath)
}

func (r *WorkspaceService) BundlePath(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error) {
	if err := packagingdomain.ValidateTarget(target); err != nil {
		return "", err
	}
	if err := packagingContextError(ctx); err != nil {
		return "", err
	}
	ctx, unlock, err := r.workspace.LockWorkspace(ctx, workspacePath)
	if err != nil {
		return "", err
	}
	defer unlock()

	return r.adapter.BundlePathLocked(ctx, workspacePath, target)
}

func (r *WorkspaceService) BundleFilePath(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error) {
	if err := packagingdomain.ValidateTarget(target); err != nil {
		return "", err
	}
	if err := packagingContextError(ctx); err != nil {
		return "", err
	}
	ctx, unlock, err := r.workspace.LockWorkspace(ctx, workspacePath)
	if err != nil {
		return "", err
	}
	defer unlock()

	bundlePath, err := r.adapter.BundlePathLocked(ctx, workspacePath, target)
	if err != nil {
		return "", err
	}
	if r.workspace.HasWorkspaceStore() {
		return r.workspace.ManagedBundlePath(workspacePath, target), nil
	}
	return bundlePath, nil
}

func (r *WorkspaceService) BundleContent(ctx context.Context, workspacePath string, target packagingdomain.Target) ([]byte, error) {
	if err := packagingdomain.ValidateTarget(target); err != nil {
		return nil, err
	}
	if err := packagingContextError(ctx); err != nil {
		return nil, err
	}
	ctx, unlock, err := r.workspace.LockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return r.adapter.BundleContentLocked(ctx, workspacePath, target, r.Pack)
}

func (r *WorkspaceService) PackageMetadata(ctx context.Context, workspacePath string) (map[string]any, error) {
	if err := packagingContextError(ctx); err != nil {
		return nil, err
	}
	ctx, unlock, err := r.workspace.LockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return r.adapter.PackageMetadataLocked(ctx, workspacePath)
}

func (r *WorkspaceService) preparePublishedRelease(ctx context.Context, input packagingdomain.PublishRequest) (publishedReleaseMaterial, error) {
	if err := packagingdomain.ValidateTarget(input.Target); err != nil {
		return publishedReleaseMaterial{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return publishedReleaseMaterial{}, err
	}
	prepared, err := r.Prepare(ctx, packagingdomain.PrepareRequest{
		IdempotencyKey: input.IdempotencyKey,
		Source:         input.Source,
		Target:         input.Target,
		Metadata:       input.Metadata,
		Options:        input.Options,
	})
	if err != nil {
		return publishedReleaseMaterial{}, err
	}

	workspacePath := prepared.WorkspacePath
	ctx, unlock, err := r.workspace.LockWorkspace(ctx, workspacePath)
	if err != nil {
		return publishedReleaseMaterial{}, err
	}

	packResult, err := r.adapter.PackLocked(ctx, packagingdomain.PackRequest{
		WorkspacePath: workspacePath,
		Target:        input.Target,
	}, r.bundleRules)
	if err != nil {
		unlock()
		return publishedReleaseMaterial{}, err
	}

	material := publishedReleaseMaterial{
		packResponse: packResult.Response,
		cleanup: func() {
			packResult.Cleanup()()
			unlock()
		},
	}
	if len(packResult.Response.Errors) > 0 {
		return material, nil
	}

	metadata, err := packaginginfra.ReadWorkspacePackageMetadataFile(workspacePath, packResult.MaterializedPath())
	if err != nil {
		material.cleanup()
		return publishedReleaseMaterial{}, err
	}
	material.metadata = metadata
	material.tarballPath = packResult.BundlePath()
	return material, nil
}

func (r *WorkspaceService) PruneExpiredWorkspaces(ctx context.Context, ttl time.Duration) error {
	if err := packagingContextError(ctx); err != nil {
		return err
	}
	return r.adapter.PruneExpired(ctx, ttl)
}
