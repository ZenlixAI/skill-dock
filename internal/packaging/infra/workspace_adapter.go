package infra

import (
	"context"
	"os"
	"path/filepath"
	"time"

	catalog "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
	persist "skill_registry/internal/platform/persistence"
)

type WorkspaceAdapter interface {
	Prepare(ctx context.Context, input packagingdomain.PrepareRequest, bundleRules packagingdomain.BundleRules) (packagingdomain.PrepareResponse, error)
	ValidateLocked(ctx context.Context, input packagingdomain.ValidateRequest, bundleRules packagingdomain.BundleRules) (packagingdomain.ValidateResponse, error)
	PackLocked(ctx context.Context, input packagingdomain.PackRequest, bundleRules packagingdomain.BundleRules) (PackExecutionResult, error)
	CleanupLocked(ctx context.Context, workspaceHandle string) error
	BundleHandleLocked(ctx context.Context, workspaceHandle string, target packagingdomain.Target) (string, error)
	BundleContentLocked(ctx context.Context, workspaceHandle string, target packagingdomain.Target, pack func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error)
	PackageMetadataLocked(ctx context.Context, workspaceHandle string) (map[string]any, error)
	PruneExpired(ctx context.Context, ttl time.Duration) error
}

type managedWorkspaceAdapter struct {
	workspace *packagingWorkspaceBacking
}

type ephemeralWorkspaceAdapter struct {
	workspace *packagingWorkspaceBacking
}

func NewWorkspaceAdapter(workspace *WorkspaceBacking) WorkspaceAdapter {
	if workspace != nil && workspace.workspaceStore != nil {
		return &managedWorkspaceAdapter{workspace: workspace}
	}
	return &ephemeralWorkspaceAdapter{workspace: workspace}
}

func (a *managedWorkspaceAdapter) Prepare(ctx context.Context, input packagingdomain.PrepareRequest, bundleRules packagingdomain.BundleRules) (packagingdomain.PrepareResponse, error) {
	workspacePath, err := a.workspace.workspaceSource.prepare(input)
	if err != nil {
		return packagingdomain.PrepareResponse{}, err
	}

	outcome, err := bundleRules.PrepareWorkspace(workspacePath, input)
	if err != nil {
		return packagingdomain.PrepareResponse{}, err
	}

	handle, _, persistErr := a.workspace.persistManagedWorkspace(ctx, workspacePath)
	if persistErr != nil {
		_ = os.RemoveAll(workspacePath)
		return packagingdomain.PrepareResponse{}, persistErr
	}
	_ = os.RemoveAll(workspacePath)

	return prepareResponseFromOutcome(handle, input, outcome), nil
}

func (a *ephemeralWorkspaceAdapter) Prepare(_ context.Context, input packagingdomain.PrepareRequest, bundleRules packagingdomain.BundleRules) (packagingdomain.PrepareResponse, error) {
	workspacePath, err := a.workspace.workspaceSource.prepare(input)
	if err != nil {
		return packagingdomain.PrepareResponse{}, err
	}

	outcome, err := bundleRules.PrepareWorkspace(workspacePath, input)
	if err != nil {
		return packagingdomain.PrepareResponse{}, err
	}

	return prepareResponseFromOutcome(workspacePath, input, outcome), nil
}

func prepareResponseFromOutcome(workspaceHandle string, input packagingdomain.PrepareRequest, outcome packagingdomain.PreparedWorkspaceOutcome) packagingdomain.PrepareResponse {
	return packagingdomain.PrepareResponse{
		WorkspaceHandle: workspaceHandle,
		PackageName:     input.Target.PackageName,
		Version:         input.Target.Version,
		Warnings:        outcome.Warnings,
		WarningDetails:  packagingdomain.WarningDetails(outcome.Warnings),
		ImportReport:    outcome.ImportReport,
		Errors:          []string{},
	}
}

func (a *managedWorkspaceAdapter) ValidateLocked(ctx context.Context, input packagingdomain.ValidateRequest, bundleRules packagingdomain.BundleRules) (packagingdomain.ValidateResponse, error) {
	record, err := a.workspace.loadManagedWorkspaceRecord(ctx, input.WorkspaceHandle)
	if err != nil {
		return packagingdomain.ValidateResponse{}, err
	}
	if cached, ok := cachedValidationResult(record.ValidationCache, input.Target); ok {
		return validateResponseFromResult(input, cached, record.WorkspacePath), nil
	}

	result, err := bundleRules.Validate(record.WorkspacePath, input.Target)
	if err != nil {
		return packagingdomain.ValidateResponse{}, err
	}
	if saveErr := a.workspace.workspaceStore.SaveValidationCache(ctx, input.WorkspaceHandle, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
		return packagingdomain.ValidateResponse{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
	}
	return validateResponseFromResult(input, result, record.WorkspacePath), nil
}

func (a *managedWorkspaceAdapter) PackLocked(ctx context.Context, input packagingdomain.PackRequest, bundleRules packagingdomain.BundleRules) (PackExecutionResult, error) {
	return packManagedWorkspace(ctx, a.workspace, bundleRules, input)
}

func (a *ephemeralWorkspaceAdapter) ValidateLocked(ctx context.Context, input packagingdomain.ValidateRequest, bundleRules packagingdomain.BundleRules) (packagingdomain.ValidateResponse, error) {
	workspacePath, cleanup, err := a.workspace.materializeWorkspace(ctx, input.WorkspaceHandle)
	if err != nil {
		return packagingdomain.ValidateResponse{}, err
	}
	defer cleanup()

	result, err := bundleRules.Validate(workspacePath, input.Target)
	if err != nil {
		return packagingdomain.ValidateResponse{}, err
	}

	return packagingdomain.ValidateResponse{
		OK:              len(result.Errors) == 0,
		WorkspaceHandle: input.WorkspaceHandle,
		PackageName:     input.Target.PackageName,
		Version:         input.Target.Version,
		ManifestDigest:  result.ManifestDigest,
		Components:      result.Components,
		Warnings:        result.Warnings,
		WarningDetails:  packagingdomain.WarningDetails(result.Warnings),
		ImportReport:    packagingdomain.ReadImportReport(workspacePath, result.Warnings),
		Errors:          result.Errors,
	}, nil
}

func (a *ephemeralWorkspaceAdapter) PackLocked(ctx context.Context, input packagingdomain.PackRequest, bundleRules packagingdomain.BundleRules) (PackExecutionResult, error) {
	return packEphemeralWorkspace(ctx, a.workspace, bundleRules, input)
}

func (a *managedWorkspaceAdapter) CleanupLocked(ctx context.Context, workspaceHandle string) error {
	return a.cleanupLocked(ctx, workspaceHandle)
}

func (a *managedWorkspaceAdapter) cleanupLocked(ctx context.Context, workspaceHandle string) error {
	record, err := a.workspace.workspaceStore.LoadWorkspace(ctx, workspaceHandle)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return catalog.NewError(422, "invalid_request", "workspaceHandle must reference an existing packaging workspace.", map[string]any{"workspaceHandle": workspaceHandle})
		}
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	managedRoot, err := a.workspace.managedWorkspaceRootFromPath(workspaceHandle, record.WorkspacePath)
	if err != nil {
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	if err := os.RemoveAll(managedRoot); err != nil {
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	if err := a.workspace.workspaceStore.DeleteWorkspace(ctx, workspaceHandle); err != nil {
		return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
	}
	return nil
}

func (a *ephemeralWorkspaceAdapter) CleanupLocked(ctx context.Context, workspaceHandle string) error {
	return a.cleanupLocked(ctx, workspaceHandle)
}

func (a *ephemeralWorkspaceAdapter) cleanupLocked(_ context.Context, workspaceHandle string) error {
	return a.workspace.workspaceSource.cleanup(workspaceHandle)
}

func (a *managedWorkspaceAdapter) BundleHandleLocked(ctx context.Context, workspaceHandle string, target packagingdomain.Target) (string, error) {
	return a.bundleHandleLocked(ctx, workspaceHandle, target)
}

func (a *managedWorkspaceAdapter) bundleHandleLocked(ctx context.Context, workspaceHandle string, target packagingdomain.Target) (string, error) {
	_, err := a.workspace.workspaceStore.LoadBundle(ctx, workspaceHandle)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return "", catalog.NewError(422, "invalid_request", "workspaceHandle must reference an existing packaging workspace.", map[string]any{"workspaceHandle": workspaceHandle})
		}
		if err == persist.ErrWorkspaceBundleNotFound {
			return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspaceHandle": workspaceHandle})
		}
		return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	bundlePath := a.workspace.managedBundlePath(workspaceHandle, target)
	if err := a.workspace.validateManagedBundlePath(workspaceHandle, target, bundlePath); err != nil {
		return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	info, statErr := os.Stat(bundlePath)
	if statErr != nil || info.IsDir() {
		if os.IsNotExist(statErr) || statErr == nil {
			return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspaceHandle": workspaceHandle})
		}
		return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	return workspaceBundleHandle(workspaceHandle, target), nil
}

func (a *ephemeralWorkspaceAdapter) BundleHandleLocked(ctx context.Context, workspaceHandle string, target packagingdomain.Target) (string, error) {
	return a.bundleHandleLocked(ctx, workspaceHandle, target)
}

func (a *ephemeralWorkspaceAdapter) bundleHandleLocked(_ context.Context, workspaceHandle string, target packagingdomain.Target) (string, error) {
	return a.workspace.workspaceSource.bundlePath(workspaceHandle, target)
}

func (a *managedWorkspaceAdapter) BundleContentLocked(ctx context.Context, workspaceHandle string, target packagingdomain.Target, pack func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error) {
	return a.bundleContentLocked(ctx, workspaceHandle, target, pack)
}

func (a *managedWorkspaceAdapter) bundleContentLocked(ctx context.Context, workspaceHandle string, target packagingdomain.Target, pack func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error) {
	_, err := a.workspace.workspaceStore.LoadBundle(ctx, workspaceHandle)
	bundlePath := a.workspace.managedBundlePath(workspaceHandle, target)
	if err == nil {
		if err := a.workspace.validateManagedBundlePath(workspaceHandle, target, bundlePath); err != nil {
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

	packed, packErr := pack(ctx, packagingdomain.PackRequest{
		WorkspaceHandle: workspaceHandle,
		Target:          target,
	})
	if packErr != nil {
		return nil, packErr
	}
	if len(packed.Errors) > 0 {
		return nil, catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspaceHandle": workspaceHandle})
	}
	_, err = a.workspace.workspaceStore.LoadBundle(ctx, workspaceHandle)
	if err != nil {
		return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	if err := a.workspace.validateManagedBundlePath(workspaceHandle, target, bundlePath); err != nil {
		return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	bundleContent, readErr := os.ReadFile(bundlePath)
	if readErr != nil {
		return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	return bundleContent, nil
}

func (a *ephemeralWorkspaceAdapter) BundleContentLocked(ctx context.Context, workspaceHandle string, target packagingdomain.Target, pack func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error) {
	return a.bundleContentLocked(ctx, workspaceHandle, target, pack)
}

func (a *ephemeralWorkspaceAdapter) bundleContentLocked(_ context.Context, workspaceHandle string, target packagingdomain.Target, _ func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error) {
	bundlePath, err := a.workspace.workspaceSource.bundlePath(workspaceHandle, target)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(bundlePath)
}

func (a *managedWorkspaceAdapter) PackageMetadataLocked(ctx context.Context, workspaceHandle string) (map[string]any, error) {
	return a.packageMetadataLocked(ctx, workspaceHandle)
}

func (a *managedWorkspaceAdapter) packageMetadataLocked(ctx context.Context, workspaceHandle string) (map[string]any, error) {
	record, err := a.workspace.loadManagedWorkspaceRecord(ctx, workspaceHandle)
	if err != nil {
		return nil, err
	}
	if len(record.PackageMetadataJSON) > 0 {
		return decodeWorkspacePackageMetadataJSON(workspaceHandle, record.PackageMetadataJSON)
	}

	content, err := os.ReadFile(filepath.Join(record.WorkspacePath, "package.json"))
	if err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json could not be read.", map[string]any{
			"workspaceHandle": workspaceHandle,
		})
	}
	payload, err := decodeWorkspacePackageMetadataJSON(workspaceHandle, content)
	if err != nil {
		return nil, err
	}
	if saveErr := a.workspace.workspaceStore.SavePackageMetadata(ctx, workspaceHandle, content, time.Now().UTC()); saveErr != nil {
		return nil, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace package metadata cache.", nil)
	}
	return payload, nil
}

func (a *ephemeralWorkspaceAdapter) PackageMetadataLocked(ctx context.Context, workspaceHandle string) (map[string]any, error) {
	return a.packageMetadataLocked(ctx, workspaceHandle)
}

func (a *ephemeralWorkspaceAdapter) packageMetadataLocked(ctx context.Context, workspaceHandle string) (map[string]any, error) {
	materializedPath, cleanup, err := a.workspace.materializeWorkspace(ctx, workspaceHandle)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	content, err := os.ReadFile(filepath.Join(materializedPath, "package.json"))
	if err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json could not be read.", map[string]any{
			"workspaceHandle": workspaceHandle,
		})
	}

	return decodeWorkspacePackageMetadataJSON(workspaceHandle, content)
}

func (a *managedWorkspaceAdapter) PruneExpired(ctx context.Context, ttl time.Duration) error {
	return a.pruneExpired(ctx, ttl)
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
					managedRoot, rootErr := a.workspace.canonicalManagedWorkspaceRoot(current.WorkspaceID)
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

func (a *ephemeralWorkspaceAdapter) PruneExpired(ctx context.Context, ttl time.Duration) error {
	return a.pruneExpired(ctx, ttl)
}

func (a *ephemeralWorkspaceAdapter) pruneExpired(_ context.Context, ttl time.Duration) error {
	return a.workspace.workspaceSource.pruneExpiredWorkspaces(ttl)
}
