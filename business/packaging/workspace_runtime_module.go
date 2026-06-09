package packaging

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

type packagingWorkspaceRuntime struct {
	workspaceRoot   string
	workspaceSource workspaceSourceModule
	bundleRules     bundleRuleModule
	workspaceStore  persist.WorkspaceStore

	workspaceLocksMu sync.Mutex
	workspaceLocks   map[string]*workspaceLockEntry
}

func newPackagingWorkspaceRuntime(tempRoot string, store persist.WorkspaceStore) *packagingWorkspaceRuntime {
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	return &packagingWorkspaceRuntime{
		workspaceRoot:   tempRoot,
		workspaceSource: newWorkspaceSourceModule(tempRoot, 24*time.Hour),
		bundleRules:     newBundleRuleModule(),
		workspaceStore:  store,
		workspaceLocks:  map[string]*workspaceLockEntry{},
	}
}

func (r *packagingWorkspaceRuntime) Prepare(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PrepareResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return PrepareResponse{}, err
	}

	workspacePath, err := r.workspaceSource.prepare(input)
	if err != nil {
		return PrepareResponse{}, err
	}

	outcome, err := r.bundleRules.prepareWorkspace(workspacePath, input)
	if err != nil {
		return PrepareResponse{}, err
	}

	if r.workspaceStore != nil {
		handle, _, persistErr := r.persistManagedWorkspace(ctx, workspacePath)
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
	ctx, unlock, err := r.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	defer unlock()

	return r.validateLocked(ctx, input)
}

func (r *packagingWorkspaceRuntime) validateLocked(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	if r.workspaceStore != nil {
		record, err := r.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
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
		if saveErr := r.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
			return ValidateResponse{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
		}
		return validateResponseFromResult(input, result, record.WorkspacePath), nil
	}

	workspacePath, cleanup, err := r.materializeWorkspace(ctx, input.WorkspacePath)
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
	ctx, unlock, err := r.lockWorkspace(ctx, input.WorkspacePath)
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
	if r.workspaceStore != nil {
		record, err := r.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
		if err != nil {
			return packExecutionResult{}, err
		}

		result, ok := cachedValidationResult(record.ValidationCache, input.Target)
		if !ok {
			result, err = r.bundleRules.validate(record.WorkspacePath, input.Target)
			if err != nil {
				return packExecutionResult{}, err
			}
			if saveErr := r.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
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

		bundlePath := r.managedBundlePath(input.WorkspacePath, input.Target)
		if err := r.validateManagedBundlePath(input.WorkspacePath, input.Target, bundlePath); err != nil {
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
		if saveErr := r.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundlePath, time.Now().UTC()); saveErr != nil {
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

	workspacePath, cleanup, err := r.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return packExecutionResult{}, err
	}

	result, err := r.bundleRules.validate(workspacePath, input.Target)
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}
	if r.workspaceStore != nil {
		if saveErr := r.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
			cleanup()
			return packExecutionResult{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
		}
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

	bundlePath, err := r.workspaceSource.bundlePath(workspacePath, input.Target)
	if r.workspaceStore != nil {
		bundlePath = r.managedBundlePath(input.WorkspacePath, input.Target)
		if err := r.validateManagedBundlePath(input.WorkspacePath, input.Target, bundlePath); err != nil {
			cleanup()
			return packExecutionResult{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
		}
	} else if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}

	bundleBase64 := ""
	if r.workspaceStore != nil {
		bundleBase64, err = r.buildManagedBundle(workspacePath, input.WorkspacePath, input.Target, input.IncludeBundle)
	} else {
		bundleBase64, err = r.bundleRules.buildBundle(workspacePath, bundlePath, input.IncludeBundle)
	}
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}
	if r.workspaceStore != nil {
		if saveErr := r.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundlePath, time.Now().UTC()); saveErr != nil {
			cleanup()
			return packExecutionResult{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
		}
		response.BundlePath = workspaceBundleHandle(input.WorkspacePath, input.Target)
	} else {
		response.BundlePath = bundlePath
	}
	response.BundleBase64 = bundleBase64

	return packExecutionResult{
		response:         response,
		materializedPath: workspacePath,
		bundlePath:       bundlePath,
		cleanup:          cleanup,
	}, nil
}

func (r *packagingWorkspaceRuntime) lockWorkspace(ctx context.Context, workspaceID string) (context.Context, func(), error) {
	if r.workspaceStore == nil || strings.TrimSpace(workspaceID) == "" {
		return ctx, func() {}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if current, ok := ctx.Value(workspaceLockContextKey{}).(string); ok && current == workspaceID {
		return ctx, func() {}, nil
	}

	r.workspaceLocksMu.Lock()
	entry := r.workspaceLocks[workspaceID]
	if entry == nil {
		entry = &workspaceLockEntry{}
		r.workspaceLocks[workspaceID] = entry
	}
	entry.refs++
	r.workspaceLocksMu.Unlock()

	entry.mu.Lock()

	release := func() {
		entry.mu.Unlock()

		r.workspaceLocksMu.Lock()
		defer r.workspaceLocksMu.Unlock()

		entry.refs--
		if entry.refs == 0 && r.workspaceLocks[workspaceID] == entry {
			delete(r.workspaceLocks, workspaceID)
		}
	}

	return context.WithValue(ctx, workspaceLockContextKey{}, workspaceID), release, nil
}

func (r *packagingWorkspaceRuntime) buildManagedBundle(workspacePath string, workspaceID string, target Target, includeBundle bool) (string, error) {
	finalPath := r.managedBundlePath(workspaceID, target)
	if err := r.validateManagedBundlePath(workspaceID, target, finalPath); err != nil {
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
