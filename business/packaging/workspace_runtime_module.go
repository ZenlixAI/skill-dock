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
	bundleRules bundleRuleModule
}

func newPackagingWorkspaceRuntime(tempRoot string, store persist.WorkspaceStore) *packagingWorkspaceRuntime {
	return newPackagingWorkspaceRuntimeWithBacking(newPackagingWorkspaceBacking(tempRoot, store))
}

func newPackagingWorkspaceRuntimeWithBacking(workspace *packagingWorkspaceBacking) *packagingWorkspaceRuntime {
	return &packagingWorkspaceRuntime{
		workspace:   workspace,
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

	workspacePath, err := r.workspace.workspaceSource.prepare(input)
	if err != nil {
		return PrepareResponse{}, err
	}

	outcome, err := r.bundleRules.prepareWorkspace(workspacePath, input)
	if err != nil {
		return PrepareResponse{}, err
	}

	if r.workspace.workspaceStore != nil {
		handle, _, persistErr := r.workspace.persistManagedWorkspace(ctx, workspacePath)
		if persistErr != nil {
			_ = os.RemoveAll(workspacePath)
			return PrepareResponse{}, persistErr
		}
		_ = os.RemoveAll(workspacePath)
		workspacePath = handle
	}

	return PrepareResponse{
		WorkspacePath:  workspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		Warnings:       outcome.Warnings,
		WarningDetails: warningDetails(outcome.Warnings),
		ImportReport:   outcome.ImportReport,
		Errors:         []string{},
	}, nil
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
	if r.workspace.workspaceStore != nil {
		record, err := r.workspace.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
		if err != nil {
			return ValidateResponse{}, err
		}
		if cached, ok := cachedValidationResult(record.ValidationCache, input.Target); ok {
			return validateResponseFromResult(input, cached, record.WorkspacePath), nil
		}

		result, err := r.bundleRules.validate(record.WorkspacePath, input.Target)
		if err != nil {
			return ValidateResponse{}, err
		}
		if saveErr := r.workspace.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
			return ValidateResponse{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
		}
		return validateResponseFromResult(input, result, record.WorkspacePath), nil
	}

	workspacePath, cleanup, err := r.workspace.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	defer cleanup()

	result, err := r.bundleRules.validate(workspacePath, input.Target)
	if err != nil {
		return ValidateResponse{}, err
	}

	return ValidateResponse{
		OK:             len(result.Errors) == 0,
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     result.Components,
		Warnings:       result.Warnings,
		WarningDetails: warningDetails(result.Warnings),
		ImportReport:   readImportReport(workspacePath, result.Warnings),
		Errors:         result.Errors,
	}, nil
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
