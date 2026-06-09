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

func (r *packagingWorkspaceRuntime) Cleanup(ctx context.Context, workspacePath string) error {
	if err := packagingContextError(ctx); err != nil {
		return err
	}
	ctx, unlock, err := r.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return err
	}
	defer unlock()

	if r.workspace.workspaceStore != nil {
		record, err := r.workspace.workspaceStore.LoadWorkspace(ctx, workspacePath)
		if err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		managedRoot, err := r.workspace.managedWorkspaceRootFromPath(workspacePath, record.WorkspacePath)
		if err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		if err := os.RemoveAll(managedRoot); err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		if err := r.workspace.workspaceStore.DeleteWorkspace(ctx, workspacePath); err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		return nil
	}
	return r.workspace.workspaceSource.cleanup(workspacePath)
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

	if r.workspace.workspaceStore != nil {
		bundlePath, err := r.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
		if err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return "", catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			if err == persist.ErrWorkspaceBundleNotFound {
				return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
			}
			return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		if err := r.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
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
	return r.workspace.workspaceSource.bundlePath(workspacePath, target)
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

	if r.workspace.workspaceStore != nil {
		bundlePath, err := r.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
		if err == nil {
			if err := r.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
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
		packed, packErr := r.Pack(ctx, PackRequest{
			WorkspacePath: workspacePath,
			Target:        target,
		})
		if packErr != nil {
			return nil, packErr
		}
		if len(packed.Errors) > 0 {
			return nil, catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
		}
		bundlePath, err = r.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
		if err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		if err := r.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		bundleContent, readErr := os.ReadFile(bundlePath)
		if readErr != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		return bundleContent, nil
	}

	bundlePath, err := r.workspace.workspaceSource.bundlePath(workspacePath, target)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(bundlePath)
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

	if r.workspace.workspaceStore != nil {
		record, err := r.workspace.loadManagedWorkspaceRecord(ctx, workspacePath)
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
		if saveErr := r.workspace.workspaceStore.SavePackageMetadata(ctx, workspacePath, content, time.Now().UTC()); saveErr != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace package metadata cache.", nil)
		}
		return payload, nil
	}

	materializedPath, cleanup, err := r.workspace.materializeWorkspace(ctx, workspacePath)
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
	if r.workspace.workspaceStore != nil && ttl > 0 {
		cutoff := time.Now().UTC().Add(-ttl)
		expiredRecords, err := r.workspace.workspaceStore.ListExpired(ctx, cutoff)
		if err != nil {
			return err
		}
		for _, record := range expiredRecords {
			if err := packagingContextError(ctx); err != nil {
				return err
			}
			lockCtx, unlock, lockErr := r.workspace.lockWorkspace(ctx, record.WorkspaceID)
			if lockErr != nil {
				return lockErr
			}
			current, loadErr := r.workspace.workspaceStore.LoadWorkspace(lockCtx, record.WorkspaceID)
			if loadErr == nil {
				if current.UpdatedAt.Before(cutoff) {
					managedRoot, rootErr := r.workspace.managedWorkspaceRootFromPath(current.WorkspaceID, current.WorkspacePath)
					if rootErr != nil {
						unlock()
						return rootErr
					}
					if err := os.RemoveAll(managedRoot); err != nil {
						unlock()
						return err
					}
					if err := r.workspace.workspaceStore.DeleteWorkspace(lockCtx, record.WorkspaceID); err != nil {
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
	return r.workspace.workspaceSource.pruneExpiredWorkspaces(ttl)
}
