package packaging

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/business/catalog"
	artifactadapter "skill_registry/foundation/artifact"
	"skill_registry/foundation/persist"
)

type Service struct {
	workspace        *packagingWorkspaceBacking
	workspaceRuntime *packagingWorkspaceRuntime
	publishWorkflow  *publishedReleaseWorkflow
}

type packExecutionResult struct {
	response         PackResponse
	materializedPath string
	bundlePath       string
	cleanup          func()
}

type releaseRecordCreator interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error)
}

type packagePublisher interface {
	PublishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) error
}

type publishedReleaseWorkflow struct {
	releaseRecords releaseRecordCreator
	publisher      packagePublisher
}

func NewService(tempRoot string) *Service {
	return NewServiceWithWorkspaceStore(tempRoot, nil)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	workspace := newPackagingWorkspaceBacking(tempRoot, store)
	return &Service{
		workspace:        workspace,
		workspaceRuntime: newPackagingWorkspaceRuntimeWithBacking(workspace),
		publishWorkflow:  &publishedReleaseWorkflow{},
	}
}

func (s *Service) ConfigurePublishWorkflow(repo releaseRecordCreator, publisher packagePublisher) {
	s.publishWorkflow.releaseRecords = repo
	s.publishWorkflow.publisher = publisher
}

func (s *Service) Prepare(input PrepareRequest) (PrepareResponse, error) {
	return s.PrepareContext(context.Background(), input)
}

func (s *Service) PrepareContext(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	return s.workspaceRuntime.Prepare(ctx, input)
}

func (s *Service) Validate(input ValidateRequest) (ValidateResponse, error) {
	return s.ValidateContext(context.Background(), input)
}

func (s *Service) ValidateContext(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	return s.workspaceRuntime.Validate(ctx, input)
}

func (s *Service) Pack(input PackRequest) (PackResponse, error) {
	return s.PackContext(context.Background(), input)
}

func (s *Service) PackContext(ctx context.Context, input PackRequest) (PackResponse, error) {
	return s.workspaceRuntime.Pack(ctx, input)
}

func (s *Service) Cleanup(workspacePath string) error {
	return s.CleanupContext(context.Background(), workspacePath)
}

func (s *Service) CleanupContext(ctx context.Context, workspacePath string) error {
	if err := packagingContextError(ctx); err != nil {
		return err
	}
	ctx, unlock, err := s.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return err
	}
	defer unlock()

	if s.workspace.workspaceStore != nil {
		record, err := s.workspace.workspaceStore.LoadWorkspace(ctx, workspacePath)
		if err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		managedRoot, err := s.workspace.managedWorkspaceRootFromPath(workspacePath, record.WorkspacePath)
		if err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		if err := os.RemoveAll(managedRoot); err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		if err := s.workspace.workspaceStore.DeleteWorkspace(ctx, workspacePath); err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		return nil
	}
	return s.workspace.workspaceSource.cleanup(workspacePath)
}

func (s *Service) BundlePath(workspacePath string, target Target) (string, error) {
	return s.BundlePathContext(context.Background(), workspacePath, target)
}

func (s *Service) BundlePathContext(ctx context.Context, workspacePath string, target Target) (string, error) {
	if err := validateTarget(target); err != nil {
		return "", err
	}
	if err := packagingContextError(ctx); err != nil {
		return "", err
	}
	ctx, unlock, err := s.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return "", err
	}
	defer unlock()

	if s.workspace.workspaceStore != nil {
		bundlePath, err := s.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
		if err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return "", catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			if err == persist.ErrWorkspaceBundleNotFound {
				return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
			}
			return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		if err := s.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
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
	return s.workspace.workspaceSource.bundlePath(workspacePath, target)
}

func (s *Service) BundleContent(workspacePath string, target Target) ([]byte, error) {
	return s.BundleContentContext(context.Background(), workspacePath, target)
}

func (s *Service) BundleContentContext(ctx context.Context, workspacePath string, target Target) ([]byte, error) {
	if err := validateTarget(target); err != nil {
		return nil, err
	}
	if err := packagingContextError(ctx); err != nil {
		return nil, err
	}
	ctx, unlock, err := s.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	if s.workspace.workspaceStore != nil {
		bundlePath, err := s.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
		if err == nil {
			if err := s.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
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
		packed, packErr := s.PackContext(ctx, PackRequest{
			WorkspacePath: workspacePath,
			Target:        target,
		})
		if packErr != nil {
			return nil, packErr
		}
		if len(packed.Errors) > 0 {
			return nil, catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
		}
		bundlePath, err = s.workspace.workspaceStore.LoadBundle(ctx, workspacePath)
		if err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		if err := s.workspace.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		bundleContent, readErr := os.ReadFile(bundlePath)
		if readErr != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		return bundleContent, nil
	}
	bundlePath, err := s.workspace.workspaceSource.bundlePath(workspacePath, target)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(bundlePath)
}

func (s *Service) PackageMetadata(workspacePath string) (map[string]any, error) {
	return s.PackageMetadataContext(context.Background(), workspacePath)
}

func (s *Service) PackageMetadataContext(ctx context.Context, workspacePath string) (map[string]any, error) {
	if err := packagingContextError(ctx); err != nil {
		return nil, err
	}
	ctx, unlock, err := s.workspace.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	if s.workspace.workspaceStore != nil {
		record, err := s.workspace.loadManagedWorkspaceRecord(ctx, workspacePath)
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
		if saveErr := s.workspace.workspaceStore.SavePackageMetadata(ctx, workspacePath, content, time.Now().UTC()); saveErr != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace package metadata cache.", nil)
		}
		return payload, nil
	}

	materializedPath, cleanup, err := s.workspace.materializeWorkspace(ctx, workspacePath)
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

	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json is invalid JSON.", map[string]any{
			"workspacePath": workspacePath,
		})
	}
	return payload, nil
}

func (s *Service) Publish(input PublishRequest) (PublishResponse, error) {
	return s.PublishContext(context.Background(), input)
}

func (s *Service) PublishContext(ctx context.Context, input PublishRequest) (PublishResponse, error) {
	if err := packagingContextError(ctx); err != nil {
		return PublishResponse{}, err
	}
	if s.publishWorkflow == nil || s.publishWorkflow.releaseRecords == nil || s.publishWorkflow.publisher == nil {
		return PublishResponse{}, catalog.NewError(500, "internal_error", "Publish workflow is not configured.", nil)
	}
	ctx, unlock, err := s.workspace.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return PublishResponse{}, err
	}
	defer unlock()

	packResult, err := s.packWorkspaceLocked(ctx, PackRequest{
		WorkspacePath: input.WorkspacePath,
		Target:        input.Target,
	})
	if err != nil {
		return PublishResponse{}, err
	}
	defer packResult.cleanup()
	packResponse := packResult.response
	if len(packResponse.Errors) > 0 {
		return PublishResponse{}, catalog.NewError(422, "bundle_validation_failed", "Bundle validation failed before publish.", map[string]any{
			"packageName": input.Target.PackageName,
			"version":     input.Target.Version,
			"errors":      packResponse.Errors,
		})
	}

	releaseInput := catalog.ReleaseRecordInput{
		ManifestDigest: stringPtr(packResponse.ManifestDigest),
		Components:     packResponse.Components,
	}
	releaseResult, err := s.publishWorkflow.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if err == nil {
		return publishResponseFromPack(packResponse, releaseResult, false), nil
	}

	releaseErr := packagingCatalogError(err)
	if releaseErr.Code != "registry_version_not_found" {
		return PublishResponse{}, releaseErr
	}

	packageMetadata, err := readWorkspacePackageMetadataFile(input.WorkspacePath, packResult.materializedPath)
	if err != nil {
		return PublishResponse{}, err
	}
	tarball, err := os.ReadFile(packResult.bundlePath)
	if err != nil {
		return PublishResponse{}, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}

	publishErr := s.publishWorkflow.publisher.PublishPackage(ctx, artifactadapter.NPMPublishInput{
		PackageName: input.Target.PackageName,
		Version:     input.Target.Version,
		Description: packageMetadata.Description,
		DisplayName: packageMetadata.Skill.DisplayName,
		SourceType:  packageMetadata.Skill.SourceType,
		Tarball:     tarball,
	})
	if publishErr != nil {
		catalogErr := packagingCatalogError(publishErr)
		if catalogErr.Code == "registry_version_conflict" {
			releaseResult, err = s.publishWorkflow.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
			if err == nil {
				return publishResponseFromPack(packResponse, releaseResult, false), nil
			}
		}
		return PublishResponse{}, catalogErr
	}

	releaseResult, err = s.publishWorkflow.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if err != nil {
		return PublishResponse{}, packagingCatalogError(err)
	}
	return publishResponseFromPack(packResponse, releaseResult, true), nil
}

func (s *Service) packWorkspaceLocked(ctx context.Context, input PackRequest) (packExecutionResult, error) {
	return s.workspaceRuntime.packLocked(ctx, input)
}

func (s *Service) PruneExpiredWorkspaces(ttl time.Duration) error {
	if s.workspace.workspaceStore != nil && ttl > 0 {
		cutoff := time.Now().UTC().Add(-ttl)
		expiredRecords, err := s.workspace.workspaceStore.ListExpired(context.Background(), cutoff)
		if err != nil {
			return err
		}
		for _, record := range expiredRecords {
			lockCtx, unlock, lockErr := s.workspace.lockWorkspace(context.Background(), record.WorkspaceID)
			if lockErr != nil {
				return lockErr
			}
			current, loadErr := s.workspace.workspaceStore.LoadWorkspace(lockCtx, record.WorkspaceID)
			if loadErr == nil {
				if current.UpdatedAt.Before(cutoff) {
					managedRoot, rootErr := s.workspace.managedWorkspaceRootFromPath(current.WorkspaceID, current.WorkspacePath)
					if rootErr != nil {
						unlock()
						return rootErr
					}
					if err := os.RemoveAll(managedRoot); err != nil {
						unlock()
						return err
					}
					if err := s.workspace.workspaceStore.DeleteWorkspace(lockCtx, record.WorkspaceID); err != nil {
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
	return s.workspace.workspaceSource.pruneExpiredWorkspaces(ttl)
}

func packagingContextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	switch err := ctx.Err(); {
	case err == nil:
		return nil
	case err == context.DeadlineExceeded:
		return catalog.NewError(504, "request_timeout", "Request context deadline exceeded during packaging.", nil)
	default:
		return catalog.NewError(499, "request_canceled", "Request context was canceled during packaging.", nil)
	}
}

type workspacePackageMetadata struct {
	Description string `json:"description"`
	Skill       struct {
		DisplayName string             `json:"displayName"`
		SourceType  catalog.SourceType `json:"sourceType"`
	} `json:"skill"`
}

func readWorkspacePackageMetadataFile(workspaceHandle string, workspacePath string) (workspacePackageMetadata, error) {
	content, err := os.ReadFile(filepath.Join(workspacePath, "package.json"))
	if err != nil {
		return workspacePackageMetadata{}, catalog.NewError(422, "invalid_request", "workspace package.json could not be read.", map[string]any{
			"workspacePath": workspaceHandle,
		})
	}

	var payload workspacePackageMetadata
	if err := json.Unmarshal(content, &payload); err != nil {
		return workspacePackageMetadata{}, catalog.NewError(422, "invalid_request", "workspace package.json is invalid JSON.", map[string]any{
			"workspacePath": workspaceHandle,
		})
	}
	return payload, nil
}

func decodeWorkspacePackageMetadataJSON(workspaceHandle string, content []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json is invalid JSON.", map[string]any{
			"workspacePath": workspaceHandle,
		})
	}
	return payload, nil
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

func publishResponseFromPack(packResponse PackResponse, releaseResult catalog.ReleaseRecordResult, publishedToRegistry bool) PublishResponse {
	return PublishResponse{
		WorkspacePath:       packResponse.WorkspacePath,
		PackageName:         packResponse.PackageName,
		Version:             packResponse.Version,
		ManifestDigest:      packResponse.ManifestDigest,
		Components:          packResponse.Components,
		PublishedToRegistry: publishedToRegistry,
		Created:             releaseResult.Created,
		IdempotentReplay:    releaseResult.IdempotentReplay,
		VersionStatus:       releaseResult.VersionStatus,
		InstallPolicy:       releaseResult.InstallPolicy,
		Warnings:            append([]string(nil), packResponse.Warnings...),
		WarningDetails:      append([]WarningDetail(nil), packResponse.WarningDetails...),
		ImportReport:        packResponse.ImportReport,
		Errors:              append([]string(nil), packResponse.Errors...),
	}
}

func packagingCatalogError(err error) *catalog.Error {
	if err == nil {
		return nil
	}
	if catalogErr, ok := err.(*catalog.Error); ok {
		return catalogErr
	}
	return catalog.NewError(500, "internal_error", "Internal server error.", nil)
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func ensureManagedWorkspace(workspacePath string, managed bool) error {
	if managed {
		return nil
	}
	return catalog.NewError(422, "invalid_request", "workspacePath must reference a managed packaging workspace.", map[string]any{"workspacePath": workspacePath})
}
