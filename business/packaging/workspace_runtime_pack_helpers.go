package packaging

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

type packExecutionResult struct {
	response         PackResponse
	materializedPath string
	bundlePath       string
	cleanup          func()
}

func packWorkspaceWithBacking(ctx context.Context, workspace *packagingWorkspaceBacking, bundleRules bundleRuleModule, input PackRequest) (packExecutionResult, error) {
	if workspace.workspaceStore != nil {
		record, err := workspace.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
		if err != nil {
			return packExecutionResult{}, err
		}

		result, ok := cachedValidationResult(record.ValidationCache, input.Target)
		if !ok {
			result, err = bundleRules.validate(record.WorkspacePath, input.Target)
			if err != nil {
				return packExecutionResult{}, err
			}
			if saveErr := workspace.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
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

		bundlePath := workspace.managedBundlePath(input.WorkspacePath, input.Target)
		if err := workspace.validateManagedBundlePath(input.WorkspacePath, input.Target, bundlePath); err != nil {
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

		bundleBase64, err := buildManagedBundle(workspace, bundleRules, record.WorkspacePath, input.WorkspacePath, input.Target, input.IncludeBundle)
		if err != nil {
			return packExecutionResult{}, err
		}
		if saveErr := workspace.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundlePath, time.Now().UTC()); saveErr != nil {
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

	workspacePath, cleanup, err := workspace.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return packExecutionResult{}, err
	}

	result, err := bundleRules.validate(workspacePath, input.Target)
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

	bundlePath, err := workspace.workspaceSource.bundlePath(workspacePath, input.Target)
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}

	bundleBase64, err := bundleRules.buildBundle(workspacePath, bundlePath, input.IncludeBundle)
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

func buildManagedBundle(workspace *packagingWorkspaceBacking, bundleRules bundleRuleModule, workspacePath string, workspaceID string, target Target, includeBundle bool) (string, error) {
	finalPath := workspace.managedBundlePath(workspaceID, target)
	if err := workspace.validateManagedBundlePath(workspaceID, target, finalPath); err != nil {
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

	bundleBase64, err := bundleRules.buildBundle(workspacePath, tempPath, includeBundle)
	if err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}
	return bundleBase64, nil
}

func reuseManagedBundle(record persist.WorkspaceRecord, input PackRequest, expectedBundlePath string) (string, bool, error) {
	if strings.TrimSpace(record.BundlePath) == "" {
		return "", false, nil
	}
	if filepath.Clean(record.BundlePath) != filepath.Clean(expectedBundlePath) {
		return "", false, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}

	info, err := os.Stat(record.BundlePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	if info.IsDir() {
		return "", false, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	if !input.IncludeBundle {
		return "", true, nil
	}

	content, err := os.ReadFile(record.BundlePath)
	if err != nil {
		return "", false, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	return base64.StdEncoding.EncodeToString(content), true, nil
}

func workspaceValidationCacheFromResult(target Target, result validationResult) persist.WorkspaceValidationCache {
	return persist.WorkspaceValidationCache{
		PackageName:    target.PackageName,
		Version:        target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     append([]string(nil), result.Components...),
		Warnings:       append([]string(nil), result.Warnings...),
		Errors:         append([]string(nil), result.Errors...),
	}
}

func cachedValidationResult(cache *persist.WorkspaceValidationCache, target Target) (validationResult, bool) {
	if cache == nil {
		return validationResult{}, false
	}
	if cache.PackageName != target.PackageName || cache.Version != target.Version {
		return validationResult{}, false
	}
	return validationResult{
		ManifestDigest: cache.ManifestDigest,
		Components:     append([]string(nil), cache.Components...),
		Warnings:       append([]string(nil), cache.Warnings...),
		Errors:         append([]string(nil), cache.Errors...),
	}, true
}

func validateResponseFromResult(input ValidateRequest, result validationResult, workspacePath string) ValidateResponse {
	return ValidateResponse{
		OK:             len(result.Errors) == 0,
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     append([]string(nil), result.Components...),
		Warnings:       append([]string(nil), result.Warnings...),
		WarningDetails: warningDetails(result.Warnings),
		ImportReport:   readImportReport(workspacePath, result.Warnings),
		Errors:         append([]string(nil), result.Errors...),
	}
}
