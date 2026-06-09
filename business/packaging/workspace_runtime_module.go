package packaging

import (
	"context"
	"os"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

type packagingWorkspaceRuntime struct {
	workspace   *packagingWorkspaceBacking
	adapter     workspaceAdapter
	bundleRules bundleRuleModule
}

type publishedReleaseMaterial struct {
	packResponse PackResponse
	metadata     workspacePackageMetadata
	tarball      []byte
	cleanup      func()
}

func newPackagingWorkspaceRuntime(tempRoot string, store persist.WorkspaceStore) *packagingWorkspaceRuntime {
	return newPackagingWorkspaceRuntimeWithBacking(newPackagingWorkspaceBacking(tempRoot, store))
}

func newPackagingWorkspaceRuntimeWithBacking(workspace *packagingWorkspaceBacking) *packagingWorkspaceRuntime {
	return &packagingWorkspaceRuntime{
		workspace:   workspace,
		adapter:     newWorkspaceAdapter(workspace),
		bundleRules: newBundleRuleModule(),
	}
}

func (r *packagingWorkspaceRuntime) Prepare(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PrepareResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return PrepareResponse{}, err
	}

	return r.adapter.prepare(ctx, input, r.bundleRules)
}

func (r *packagingWorkspaceRuntime) Validate(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return ValidateResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return ValidateResponse{}, err
	}
	ctx, unlock, err := r.workspace.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	defer unlock()

	return r.validateLocked(ctx, input)
}

func (r *packagingWorkspaceRuntime) validateLocked(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	return r.adapter.validateLocked(ctx, input, r.bundleRules)
}

func (r *packagingWorkspaceRuntime) Pack(ctx context.Context, input PackRequest) (PackResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PackResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return PackResponse{}, err
	}
	ctx, unlock, err := r.workspace.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return PackResponse{}, err
	}
	defer unlock()

	result, err := packWorkspaceWithBacking(ctx, r.workspace, r.bundleRules, input)
	if err != nil {
		return PackResponse{}, err
	}
	defer result.cleanup()

	return result.response, nil
}

func (r *packagingWorkspaceRuntime) Cleanup(ctx context.Context, workspacePath string) error {
	if err := packagingContextError(ctx); err != nil {
		return err
	}
	ctx, unlock, err := r.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return err
	}
	defer unlock()

	return r.adapter.cleanupLocked(ctx, workspacePath)
}

func (r *packagingWorkspaceRuntime) BundlePath(ctx context.Context, workspacePath string, target Target) (string, error) {
	if err := validateTarget(target); err != nil {
		return "", err
	}
	if err := packagingContextError(ctx); err != nil {
		return "", err
	}
	ctx, unlock, err := r.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return "", err
	}
	defer unlock()

	return r.adapter.bundlePathLocked(ctx, workspacePath, target)
}

func (r *packagingWorkspaceRuntime) BundleContent(ctx context.Context, workspacePath string, target Target) ([]byte, error) {
	if err := validateTarget(target); err != nil {
		return nil, err
	}
	if err := packagingContextError(ctx); err != nil {
		return nil, err
	}
	ctx, unlock, err := r.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return r.adapter.bundleContentLocked(ctx, workspacePath, target, r.Pack)
}

func (r *packagingWorkspaceRuntime) PackageMetadata(ctx context.Context, workspacePath string) (map[string]any, error) {
	if err := packagingContextError(ctx); err != nil {
		return nil, err
	}
	ctx, unlock, err := r.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	return r.adapter.packageMetadataLocked(ctx, workspacePath)
}

func (r *packagingWorkspaceRuntime) preparePublishedRelease(ctx context.Context, input PublishRequest) (publishedReleaseMaterial, error) {
	if err := validateTarget(input.Target); err != nil {
		return publishedReleaseMaterial{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return publishedReleaseMaterial{}, err
	}
	ctx, unlock, err := r.workspace.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return publishedReleaseMaterial{}, err
	}

	packResult, err := packWorkspaceWithBacking(ctx, r.workspace, r.bundleRules, PackRequest{
		WorkspacePath: input.WorkspacePath,
		Target:        input.Target,
	})
	if err != nil {
		unlock()
		return publishedReleaseMaterial{}, err
	}

	material := publishedReleaseMaterial{
		packResponse: packResult.response,
		cleanup: func() {
			packResult.cleanup()
			unlock()
		},
	}
	if len(packResult.response.Errors) > 0 {
		return material, nil
	}

	metadata, err := readWorkspacePackageMetadataFile(input.WorkspacePath, packResult.materializedPath)
	if err != nil {
		material.cleanup()
		return publishedReleaseMaterial{}, err
	}
	tarball, err := os.ReadFile(packResult.bundlePath)
	if err != nil {
		material.cleanup()
		return publishedReleaseMaterial{}, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}

	material.metadata = metadata
	material.tarball = tarball
	return material, nil
}

func (r *packagingWorkspaceRuntime) PruneExpiredWorkspaces(ctx context.Context, ttl time.Duration) error {
	if err := packagingContextError(ctx); err != nil {
		return err
	}
	return r.adapter.pruneExpired(ctx, ttl)
}
