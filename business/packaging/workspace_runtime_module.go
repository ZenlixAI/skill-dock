package packaging

import (
	"context"
	"os"
	"path/filepath"
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

	result, err := r.packLocked(ctx, input)
	if err != nil {
		return PackResponse{}, err
	}
	defer result.cleanup()

	return result.response, nil
}

func (r *packagingWorkspaceRuntime) packLocked(ctx context.Context, input PackRequest) (packExecutionResult, error) {
	if r.workspace.workspaceStore != nil {
		record, err := r.workspace.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
		if err != nil {
			return packExecutionResult{}, err
		}

		result, ok := cachedValidationResult(record.ValidationCache, input.Target)
		if !ok {
			result, err = r.bundleRules.validate(record.WorkspacePath, input.Target)
			if err != nil {
				return packExecutionResult{}, err
			}
			if saveErr := r.workspace.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
				return packExecutionResult{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
			}
		}

		response := PackResponse{
			WorkspacePath:  input.WorkspacePath,
			PackageName:    input.Target.PackageName,
			Version:        input.Target.Version,
			ManifestDigest: result.ManifestDigest,
			Components:     append([]string(nil), result.Components...),
			Warnings:       append([]string(nil), result.Warnings...),
			WarningDetails: warningDetails(result.Warnings),
			ImportReport:   readImportReport(record.WorkspacePath, result.Warnings),
			Errors:         append([]string(nil), result.Errors...),
		}
		if len(result.Errors) > 0 {
			return packExecutionResult{
				response:         response,
				materializedPath: record.WorkspacePath,
				cleanup:          func() {},
			}, nil
		}

		bundlePath := r.workspace.managedBundlePath(input.WorkspacePath, input.Target)
		if err := r.workspace.validateManagedBundlePath(input.WorkspacePath, input.Target, bundlePath); err != nil {
			return packExecutionResult{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
		}
		if bundleBase64, ok, err := reuseManagedBundle(record, input, bundlePath); err != nil {
			return packExecutionResult{}, err
		} else if ok {
			response.BundlePath = workspaceBundleHandle(input.WorkspacePath, input.Target)
			response.BundleBase64 = bundleBase64
			return packExecutionResult{
				response:         response,
				materializedPath: record.WorkspacePath,
				bundlePath:       bundlePath,
				cleanup:          func() {},
			}, nil
		}

		bundleBase64, err := r.buildManagedBundle(record.WorkspacePath, input.WorkspacePath, input.Target, input.IncludeBundle)
		if err != nil {
			return packExecutionResult{}, err
		}
		if saveErr := r.workspace.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundlePath, time.Now().UTC()); saveErr != nil {
			return packExecutionResult{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
		}
		response.BundlePath = workspaceBundleHandle(input.WorkspacePath, input.Target)
		response.BundleBase64 = bundleBase64
		return packExecutionResult{
			response:         response,
			materializedPath: record.WorkspacePath,
			bundlePath:       bundlePath,
			cleanup:          func() {},
		}, nil
	}

	workspacePath, cleanup, err := r.workspace.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return packExecutionResult{}, err
	}

	result, err := r.bundleRules.validate(workspacePath, input.Target)
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}
	response := PackResponse{
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     result.Components,
		Warnings:       result.Warnings,
		WarningDetails: warningDetails(result.Warnings),
		ImportReport:   readImportReport(workspacePath, result.Warnings),
		Errors:         result.Errors,
	}
	if len(result.Errors) > 0 {
		return packExecutionResult{
			response:         response,
			materializedPath: workspacePath,
			cleanup:          cleanup,
		}, nil
	}

	bundlePath, err := r.workspace.workspaceSource.bundlePath(workspacePath, input.Target)
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}

	bundleBase64, err := r.bundleRules.buildBundle(workspacePath, bundlePath, input.IncludeBundle)
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}
	response.BundlePath = bundlePath
	response.BundleBase64 = bundleBase64

	return packExecutionResult{
		response:         response,
		materializedPath: workspacePath,
		bundlePath:       bundlePath,
		cleanup:          cleanup,
	}, nil
}

func (r *packagingWorkspaceRuntime) buildManagedBundle(workspacePath string, workspaceID string, target Target, includeBundle bool) (string, error) {
	finalPath := r.workspace.managedBundlePath(workspaceID, target)
	if err := r.workspace.validateManagedBundlePath(workspaceID, target, finalPath); err != nil {
		return "", catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}

	bundleDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return "", catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}

	tempFile, err := os.CreateTemp(bundleDir, "."+filepath.Base(finalPath)+".tmp-*")
	if err != nil {
		return "", catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}
	tempPath := tempFile.Name()
	if closeErr := tempFile.Close(); closeErr != nil {
		_ = os.Remove(tempPath)
		return "", catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}
	defer func() {
		_ = os.Remove(tempPath)
	}()

	bundleBase64, err := r.bundleRules.buildBundle(workspacePath, tempPath, includeBundle)
	if err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}
	return bundleBase64, nil
}
