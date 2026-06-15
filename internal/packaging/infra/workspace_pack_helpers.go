package infra

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
	packagingdomain "skill_registry/internal/packaging/domain"
)

type PackExecutionResult struct {
	Response         packagingdomain.PackResponse
	materializedPath string
	bundlePath       string
	cleanup          func()
}

func (r PackExecutionResult) MaterializedPath() string { return r.materializedPath }
func (r PackExecutionResult) BundlePath() string       { return r.bundlePath }
func (r PackExecutionResult) Cleanup() func()          { return r.cleanup }

func packManagedWorkspace(ctx context.Context, workspace *packagingWorkspaceBacking, bundleRules packagingdomain.BundleRules, input packagingdomain.PackRequest) (PackExecutionResult, error) {
	record, err := workspace.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
	if err != nil {
		return PackExecutionResult{}, err
	}

	result, ok := cachedValidationResult(record.ValidationCache, input.Target)
	if !ok {
		result, err = bundleRules.Validate(record.WorkspacePath, input.Target)
		if err != nil {
			return PackExecutionResult{}, err
		}
		if saveErr := workspace.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
			return PackExecutionResult{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
		}
	}

	response := packagingdomain.PackResponse{
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     append([]string(nil), result.Components...),
		Warnings:       append([]string(nil), result.Warnings...),
		WarningDetails: packagingdomain.WarningDetails(result.Warnings),
		ImportReport:   packagingdomain.ReadImportReport(record.WorkspacePath, result.Warnings),
		Errors:         append([]string(nil), result.Errors...),
	}
	if len(result.Errors) > 0 {
		return PackExecutionResult{
			Response:         response,
			materializedPath: record.WorkspacePath,
			cleanup:          func() {},
		}, nil
	}

	bundlePath := workspace.managedBundlePath(input.WorkspacePath, input.Target)
	if err := workspace.validateManagedBundlePath(input.WorkspacePath, input.Target, bundlePath); err != nil {
		return PackExecutionResult{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}
	if bundleBase64, ok, err := reuseManagedBundle(record, input, bundlePath); err != nil {
		return PackExecutionResult{}, err
	} else if ok {
		response.BundlePath = workspaceBundleHandle(input.WorkspacePath, input.Target)
		response.BundleBase64 = bundleBase64
		return PackExecutionResult{
			Response:         response,
			materializedPath: record.WorkspacePath,
			bundlePath:       bundlePath,
			cleanup:          func() {},
		}, nil
	}

	bundleBase64, err := buildManagedBundle(workspace, bundleRules, record.WorkspacePath, input.WorkspacePath, input.Target, input.IncludeBundle)
	if err != nil {
		return PackExecutionResult{}, err
	}
	if saveErr := workspace.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundlePath, time.Now().UTC()); saveErr != nil {
		return PackExecutionResult{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}
	response.BundlePath = workspaceBundleHandle(input.WorkspacePath, input.Target)
	response.BundleBase64 = bundleBase64
	return PackExecutionResult{
		Response:         response,
		materializedPath: record.WorkspacePath,
		bundlePath:       bundlePath,
		cleanup:          func() {},
	}, nil
}

func packEphemeralWorkspace(ctx context.Context, workspace *packagingWorkspaceBacking, bundleRules packagingdomain.BundleRules, input packagingdomain.PackRequest) (PackExecutionResult, error) {
	workspacePath, cleanup, err := workspace.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return PackExecutionResult{}, err
	}

	result, err := bundleRules.Validate(workspacePath, input.Target)
	if err != nil {
		cleanup()
		return PackExecutionResult{}, err
	}

	response := packagingdomain.PackResponse{
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     result.Components,
		Warnings:       result.Warnings,
		WarningDetails: packagingdomain.WarningDetails(result.Warnings),
		ImportReport:   packagingdomain.ReadImportReport(workspacePath, result.Warnings),
		Errors:         result.Errors,
	}
	if len(result.Errors) > 0 {
		return PackExecutionResult{
			Response:         response,
			materializedPath: workspacePath,
			cleanup:          cleanup,
		}, nil
	}

	bundlePath, err := workspace.workspaceSource.bundlePath(workspacePath, input.Target)
	if err != nil {
		cleanup()
		return PackExecutionResult{}, err
	}

	bundleBase64, err := bundleRules.BuildBundle(workspacePath, bundlePath, input.IncludeBundle)
	if err != nil {
		cleanup()
		return PackExecutionResult{}, err
	}
	response.BundlePath = bundlePath
	response.BundleBase64 = bundleBase64

	return PackExecutionResult{
		Response:         response,
		materializedPath: workspacePath,
		bundlePath:       bundlePath,
		cleanup:          cleanup,
	}, nil
}

func buildManagedBundle(workspace *packagingWorkspaceBacking, bundleRules packagingdomain.BundleRules, workspacePath string, workspaceID string, target packagingdomain.Target, includeBundle bool) (string, error) {
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

	bundleBase64, err := bundleRules.BuildBundle(workspacePath, tempPath, includeBundle)
	if err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}
	return bundleBase64, nil
}

func reuseManagedBundle(record persist.WorkspaceRecord, input packagingdomain.PackRequest, expectedBundlePath string) (string, bool, error) {
	if strings.TrimSpace(record.BundlePath) == "" {
		return "", false, nil
	}

	info, err := os.Stat(expectedBundlePath)
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

	content, err := os.ReadFile(expectedBundlePath)
	if err != nil {
		return "", false, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}
	return base64.StdEncoding.EncodeToString(content), true, nil
}

func workspaceValidationCacheFromResult(target packagingdomain.Target, result packagingdomain.ValidationResult) persist.WorkspaceValidationCache {
	return persist.WorkspaceValidationCache{
		PackageName:    target.PackageName,
		Version:        target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     append([]string(nil), result.Components...),
		Warnings:       append([]string(nil), result.Warnings...),
		Errors:         append([]string(nil), result.Errors...),
	}
}

func cachedValidationResult(cache *persist.WorkspaceValidationCache, target packagingdomain.Target) (packagingdomain.ValidationResult, bool) {
	if cache == nil {
		return packagingdomain.ValidationResult{}, false
	}
	if cache.PackageName != target.PackageName || cache.Version != target.Version {
		return packagingdomain.ValidationResult{}, false
	}
	return packagingdomain.ValidationResult{
		ManifestDigest: cache.ManifestDigest,
		Components:     append([]string(nil), cache.Components...),
		Warnings:       append([]string(nil), cache.Warnings...),
		Errors:         append([]string(nil), cache.Errors...),
	}, true
}

func validateResponseFromResult(input packagingdomain.ValidateRequest, result packagingdomain.ValidationResult, workspacePath string) packagingdomain.ValidateResponse {
	return packagingdomain.ValidateResponse{
		OK:             len(result.Errors) == 0,
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     append([]string(nil), result.Components...),
		Warnings:       append([]string(nil), result.Warnings...),
		WarningDetails: packagingdomain.WarningDetails(result.Warnings),
		ImportReport:   packagingdomain.ReadImportReport(workspacePath, result.Warnings),
		Errors:         append([]string(nil), result.Errors...),
	}
}
