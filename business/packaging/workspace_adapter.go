package packaging

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

type workspaceAdapter interface {
	prepare(ctx context.Context, input PrepareRequest, bundleRules bundleRuleModule) (PrepareResponse, error)
	validateLocked(ctx context.Context, input ValidateRequest, bundleRules bundleRuleModule) (ValidateResponse, error)
	packLocked(ctx context.Context, input PackRequest, bundleRules bundleRuleModule) (packExecutionResult, error)
	cleanupLocked(ctx context.Context, workspacePath string) error
	bundlePathLocked(ctx context.Context, workspacePath string, target Target) (string, error)
	bundleContentLocked(ctx context.Context, workspacePath string, target Target, pack func(context.Context, PackRequest) (PackResponse, error)) ([]byte, error)
	packageMetadataLocked(ctx context.Context, workspacePath string) (map[string]any, error)
	pruneExpired(ctx context.Context, ttl time.Duration) error
}

type managedWorkspaceAdapter struct {
	workspace *packagingWorkspaceBacking
}

type ephemeralWorkspaceAdapter struct {
	workspace *packagingWorkspaceBacking
}

func newWorkspaceAdapter(workspace *packagingWorkspaceBacking) workspaceAdapter {
	if workspace != nil && workspace.workspaceStore != nil {
		return &managedWorkspaceAdapter{workspace: workspace}
	}
	return &ephemeralWorkspaceAdapter{workspace: workspace}
}

func (a *managedWorkspaceAdapter) prepare(ctx context.Context, input PrepareRequest, bundleRules bundleRuleModule) (PrepareResponse, error) {
	workspacePath, err := a.workspace.workspaceSource.prepare(input)
	if err != nil {
		return PrepareResponse{}, err
	}

	outcome, err := bundleRules.prepareWorkspace(workspacePath, input)
	if err != nil {
		return PrepareResponse{}, err
	}

	handle, _, persistErr := a.workspace.persistManagedWorkspace(ctx, workspacePath)
	if persistErr != nil {
		_ = os.RemoveAll(workspacePath)
		return PrepareResponse{}, persistErr
	}
	_ = os.RemoveAll(workspacePath)

	return prepareResponseFromOutcome(handle, input, outcome), nil
}

func (a *ephemeralWorkspaceAdapter) prepare(_ context.Context, input PrepareRequest, bundleRules bundleRuleModule) (PrepareResponse, error) {
	workspacePath, err := a.workspace.workspaceSource.prepare(input)
	if err != nil {
		return PrepareResponse{}, err
	}

	outcome, err := bundleRules.prepareWorkspace(workspacePath, input)
	if err != nil {
		return PrepareResponse{}, err
	}

	return prepareResponseFromOutcome(workspacePath, input, outcome), nil
}

func prepareResponseFromOutcome(workspacePath string, input PrepareRequest, outcome preparedWorkspaceOutcome) PrepareResponse {
	return PrepareResponse{
		WorkspacePath:  workspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		Warnings:       outcome.Warnings,
		WarningDetails: warningDetails(outcome.Warnings),
		ImportReport:   outcome.ImportReport,
		Errors:         []string{},
	}
}

func (a *managedWorkspaceAdapter) validateLocked(ctx context.Context, input ValidateRequest, bundleRules bundleRuleModule) (ValidateResponse, error) {
	record, err := a.workspace.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	if cached, ok := cachedValidationResult(record.ValidationCache, input.Target); ok {
		return validateResponseFromResult(input, cached, record.WorkspacePath), nil
	}

	result, err := bundleRules.validate(record.WorkspacePath, input.Target)
	if err != nil {
		return ValidateResponse{}, err
	}
	if saveErr := a.workspace.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
		return ValidateResponse{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
	}
	return validateResponseFromResult(input, result, record.WorkspacePath), nil
}

func (a *managedWorkspaceAdapter) packLocked(ctx context.Context, input PackRequest, bundleRules bundleRuleModule) (packExecutionResult, error) {
	return packManagedWorkspace(ctx, a.workspace, bundleRules, input)
}

func (a *ephemeralWorkspaceAdapter) validateLocked(ctx context.Context, input ValidateRequest, bundleRules bundleRuleModule) (ValidateResponse, error) {
	workspacePath, cleanup, err := a.workspace.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	defer cleanup()

	result, err := bundleRules.validate(workspacePath, input.Target)
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

func (a *ephemeralWorkspaceAdapter) packLocked(ctx context.Context, input PackRequest, bundleRules bundleRuleModule) (packExecutionResult, error) {
	return packEphemeralWorkspace(ctx, a.workspace, bundleRules, input)
}

func (a *managedWorkspaceAdapter) cleanupLocked(ctx context.Context, workspacePath string) error {
	record, err := a.workspace.workspaceStore.LoadWorkspace(ctx, workspacePath)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
		}
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	managedRoot, err := a.workspace.managedWorkspaceRootFromPath(workspacePath, record.WorkspacePath)
	if err != nil {
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	if err := os.RemoveAll(managedRoot); err != nil {
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	if err := a.workspace.workspaceStore.DeleteWorkspace(ctx, workspacePath); err != nil {
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	return nil
}

func (a *ephemeralWorkspaceAdapter) cleanupLocked(_ context.Context, workspacePath string) error {
	return a.workspace.workspaceSource.cleanup(workspacePath)
}

func (a *managedWorkspaceAdapter) bundlePathLocked(ctx context.Context, workspacePath string, target Target) (string, error) {
	bundlePath, err := a.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return "", catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
		}
		if err == persist.ErrWorkspaceBundleNotFound {
			return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
		}
		return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	if err := a.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
		return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	info, statErr := os.Stat(bundlePath)
	if statErr != nil || info.IsDir() {
		if os.IsNotExist(statErr) || statErr == nil {
			return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
		}
		return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	return workspaceBundleHandle(workspacePath, target), nil
}

func (a *ephemeralWorkspaceAdapter) bundlePathLocked(_ context.Context, workspacePath string, target Target) (string, error) {
	return a.workspace.workspaceSource.bundlePath(workspacePath, target)
}

func (a *managedWorkspaceAdapter) bundleContentLocked(ctx context.Context, workspacePath string, target Target, pack func(context.Context, PackRequest) (PackResponse, error)) ([]byte, error) {
	bundlePath, err := a.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
	if err == nil {
		if err := a.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		bundleContent, readErr := os.ReadFile(bundlePath)
		if readErr == nil {
			return bundleContent, nil
		}
		if !os.IsNotExist(readErr) {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		err = persist.ErrWorkspaceBundleNotFound
	}
	if err != persist.ErrWorkspaceBundleNotFound && err != persist.ErrWorkspaceRecordNotFound {
		return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}

	packed, packErr := pack(ctx, PackRequest{
		WorkspacePath: workspacePath,
		Target:        target,
	})
	if packErr != nil {
		return nil, packErr
	}
	if len(packed.Errors) > 0 {
		return nil, catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
	}
	bundlePath, err = a.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
	if err != nil {
		return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	if err := a.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
		return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	bundleContent, readErr := os.ReadFile(bundlePath)
	if readErr != nil {
		return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	return bundleContent, nil
}

func (a *ephemeralWorkspaceAdapter) bundleContentLocked(_ context.Context, workspacePath string, target Target, _ func(context.Context, PackRequest) (PackResponse, error)) ([]byte, error) {
	bundlePath, err := a.workspace.workspaceSource.bundlePath(workspacePath, target)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(bundlePath)
}

func (a *managedWorkspaceAdapter) packageMetadataLocked(ctx context.Context, workspacePath string) (map[string]any, error) {
	record, err := a.workspace.loadManagedWorkspaceRecord(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	if len(record.PackageMetadataJSON) > 0 {
		return decodeWorkspacePackageMetadataJSON(workspacePath, record.PackageMetadataJSON)
	}

	content, err := os.ReadFile(filepath.Join(record.WorkspacePath, "package.json"))
	if err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json could not be read.", map[string]any{
			"workspacePath": workspacePath,
		})
	}
	payload, err := decodeWorkspacePackageMetadataJSON(workspacePath, content)
	if err != nil {
		return nil, err
	}
	if saveErr := a.workspace.workspaceStore.SavePackageMetadata(ctx, workspacePath, content, time.Now().UTC()); saveErr != nil {
		return nil, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace package metadata cache.", nil)
	}
	return payload, nil
}

func (a *ephemeralWorkspaceAdapter) packageMetadataLocked(ctx context.Context, workspacePath string) (map[string]any, error) {
	materializedPath, cleanup, err := a.workspace.materializeWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	content, err := os.ReadFile(filepath.Join(materializedPath, "package.json"))
	if err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json could not be read.", map[string]any{
			"workspacePath": workspacePath,
		})
	}

	return decodeWorkspacePackageMetadataJSON(workspacePath, content)
}

func (a *managedWorkspaceAdapter) pruneExpired(ctx context.Context, ttl time.Duration) error {
	if ttl > 0 {
		cutoff := time.Now().UTC().Add(-ttl)
		expiredRecords, err := a.workspace.workspaceStore.ListExpired(ctx, cutoff)
		if err != nil {
			return err
		}
		for _, record := range expiredRecords {
			if err := packagingContextError(ctx); err != nil {
				return err
			}
			lockCtx, unlock, lockErr := a.workspace.lockWorkspace(ctx, record.WorkspaceID)
			if lockErr != nil {
				return lockErr
			}
			current, loadErr := a.workspace.workspaceStore.LoadWorkspace(lockCtx, record.WorkspaceID)
			if loadErr == nil {
				if current.UpdatedAt.Before(cutoff) {
					managedRoot, rootErr := a.workspace.managedWorkspaceRootFromPath(current.WorkspaceID, current.WorkspacePath)
					if rootErr != nil {
						unlock()
						return rootErr
					}
					if err := os.RemoveAll(managedRoot); err != nil {
						unlock()
						return err
					}
					if err := a.workspace.workspaceStore.DeleteWorkspace(lockCtx, record.WorkspaceID); err != nil {
						unlock()
						return err
					}
				}
			} else if loadErr != persist.ErrWorkspaceRecordNotFound {
				unlock()
				return loadErr
			}
			unlock()
		}
	}
	return a.workspace.workspaceSource.pruneExpiredWorkspaces(ttl)
}

func (a *ephemeralWorkspaceAdapter) pruneExpired(_ context.Context, ttl time.Duration) error {
	return a.workspace.workspaceSource.pruneExpiredWorkspaces(ttl)
}
