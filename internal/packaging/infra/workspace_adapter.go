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
	CleanupLocked(ctx context.Context, workspacePath string) error
	BundlePathLocked(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error)
	BundleContentLocked(ctx context.Context, workspacePath string, target packagingdomain.Target, pack func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error)
	PackageMetadataLocked(ctx context.Context, workspacePath string) (map[string]any, error)
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

func prepareResponseFromOutcome(workspacePath string, input packagingdomain.PrepareRequest, outcome packagingdomain.PreparedWorkspaceOutcome) packagingdomain.PrepareResponse {
	return packagingdomain.PrepareResponse{
		WorkspacePath:  workspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		Warnings:       outcome.Warnings,
		WarningDetails: packagingdomain.WarningDetails(outcome.Warnings),
		ImportReport:   outcome.ImportReport,
		Errors:         []string{},
	}
}

func (a *managedWorkspaceAdapter) ValidateLocked(ctx context.Context, input packagingdomain.ValidateRequest, bundleRules packagingdomain.BundleRules) (packagingdomain.ValidateResponse, error) {
	record, err := a.workspace.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
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
	if saveErr := a.workspace.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
		return packagingdomain.ValidateResponse{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
	}
	return validateResponseFromResult(input, result, record.WorkspacePath), nil
}

func (a *managedWorkspaceAdapter) PackLocked(ctx context.Context, input packagingdomain.PackRequest, bundleRules packagingdomain.BundleRules) (PackExecutionResult, error) {
	return packManagedWorkspace(ctx, a.workspace, bundleRules, input)
}

func (a *ephemeralWorkspaceAdapter) ValidateLocked(ctx context.Context, input packagingdomain.ValidateRequest, bundleRules packagingdomain.BundleRules) (packagingdomain.ValidateResponse, error) {
	workspacePath, cleanup, err := a.workspace.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return packagingdomain.ValidateResponse{}, err
	}
	defer cleanup()

	result, err := bundleRules.Validate(workspacePath, input.Target)
	if err != nil {
		return packagingdomain.ValidateResponse{}, err
	}

	return packagingdomain.ValidateResponse{
		OK:             len(result.Errors) == 0,
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     result.Components,
		Warnings:       result.Warnings,
		WarningDetails: packagingdomain.WarningDetails(result.Warnings),
		ImportReport:   packagingdomain.ReadImportReport(workspacePath, result.Warnings),
		Errors:         result.Errors,
	}, nil
}

func (a *ephemeralWorkspaceAdapter) PackLocked(ctx context.Context, input packagingdomain.PackRequest, bundleRules packagingdomain.BundleRules) (PackExecutionResult, error) {
	return packEphemeralWorkspace(ctx, a.workspace, bundleRules, input)
}

func (a *managedWorkspaceAdapter) CleanupLocked(ctx context.Context, workspacePath string) error {
	return a.cleanupLocked(ctx, workspacePath)
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

func (a *ephemeralWorkspaceAdapter) CleanupLocked(ctx context.Context, workspacePath string) error {
	return a.cleanupLocked(ctx, workspacePath)
}

func (a *ephemeralWorkspaceAdapter) cleanupLocked(_ context.Context, workspacePath string) error {
	return a.workspace.workspaceSource.cleanup(workspacePath)
}

func (a *managedWorkspaceAdapter) BundlePathLocked(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error) {
	return a.bundlePathLocked(ctx, workspacePath, target)
}

func (a *managedWorkspaceAdapter) bundlePathLocked(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error) {
	_, err := a.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
	if err != nil {
		if err == persist.ErrWorkspaceRecordNotFound {
			return "", catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
		}
		if err == persist.ErrWorkspaceBundleNotFound {
			return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
		}
		return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	bundlePath := a.workspace.managedBundlePath(workspacePath, target)
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

func (a *ephemeralWorkspaceAdapter) BundlePathLocked(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error) {
	return a.bundlePathLocked(ctx, workspacePath, target)
}

func (a *ephemeralWorkspaceAdapter) bundlePathLocked(_ context.Context, workspacePath string, target packagingdomain.Target) (string, error) {
	return a.workspace.workspaceSource.bundlePath(workspacePath, target)
}

func (a *managedWorkspaceAdapter) BundleContentLocked(ctx context.Context, workspacePath string, target packagingdomain.Target, pack func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error) {
	return a.bundleContentLocked(ctx, workspacePath, target, pack)
}

func (a *managedWorkspaceAdapter) bundleContentLocked(ctx context.Context, workspacePath string, target packagingdomain.Target, pack func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error) {
	_, err := a.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
	bundlePath := a.workspace.managedBundlePath(workspacePath, target)
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

	packed, packErr := pack(ctx, packagingdomain.PackRequest{
		WorkspacePath: workspacePath,
		Target:        target,
	})
	if packErr != nil {
		return nil, packErr
	}
	if len(packed.Errors) > 0 {
		return nil, catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
	}
	_, err = a.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
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

func (a *ephemeralWorkspaceAdapter) BundleContentLocked(ctx context.Context, workspacePath string, target packagingdomain.Target, pack func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error) {
	return a.bundleContentLocked(ctx, workspacePath, target, pack)
}

func (a *ephemeralWorkspaceAdapter) bundleContentLocked(_ context.Context, workspacePath string, target packagingdomain.Target, _ func(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)) ([]byte, error) {
	bundlePath, err := a.workspace.workspaceSource.bundlePath(workspacePath, target)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(bundlePath)
}

func (a *managedWorkspaceAdapter) PackageMetadataLocked(ctx context.Context, workspacePath string) (map[string]any, error) {
	return a.packageMetadataLocked(ctx, workspacePath)
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

func (a *ephemeralWorkspaceAdapter) PackageMetadataLocked(ctx context.Context, workspacePath string) (map[string]any, error) {
	return a.packageMetadataLocked(ctx, workspacePath)
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
