package packaging

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"skill_registry/business/catalog"
	artifactadapter "skill_registry/foundation/artifact"
	"skill_registry/foundation/persist"
)

type Service struct {
	workspaceRoot   string
	workspaceSource workspaceSourceModule
	bundleRules     bundleRuleModule
	workspaceStore  persist.WorkspaceStore
	releaseRecords  releaseRecordCreator
	publisher       packagePublisher

	workspaceLocksMu sync.Mutex
	workspaceLocks   map[string]*workspaceLockEntry
}

type workspaceLockEntry struct {
	mu   sync.Mutex
	refs int
}

type workspaceLockContextKey struct{}

type releaseRecordCreator interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error)
}

type packagePublisher interface {
	PublishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) error
}

func NewService(tempRoot string) *Service {
	return NewServiceWithWorkspaceStore(tempRoot, nil)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	return &Service{
		workspaceRoot:   tempRoot,
		workspaceSource: newWorkspaceSourceModule(tempRoot, 24*time.Hour),
		bundleRules:     newBundleRuleModule(),
		workspaceStore:  store,
		workspaceLocks:  map[string]*workspaceLockEntry{},
	}
}

func (s *Service) ConfigurePublishWorkflow(repo releaseRecordCreator, publisher packagePublisher) {
	s.releaseRecords = repo
	s.publisher = publisher
}

func (s *Service) Prepare(input PrepareRequest) (PrepareResponse, error) {
	return s.PrepareContext(context.Background(), input)
}

func (s *Service) PrepareContext(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PrepareResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return PrepareResponse{}, err
	}

	workspacePath, err := s.workspaceSource.prepare(input)
	if err != nil {
		return PrepareResponse{}, err
	}

	outcome, err := s.bundleRules.prepareWorkspace(workspacePath, input)
	if err != nil {
		return PrepareResponse{}, err
	}

	if s.workspaceStore != nil {
		handle, _, persistErr := s.persistManagedWorkspace(ctx, workspacePath)
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

func (s *Service) Validate(input ValidateRequest) (ValidateResponse, error) {
	return s.ValidateContext(context.Background(), input)
}

func (s *Service) ValidateContext(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return ValidateResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return ValidateResponse{}, err
	}
	ctx, unlock, err := s.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	defer unlock()

	workspacePath, cleanup, err := s.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	defer cleanup()

	result, err := s.bundleRules.validate(workspacePath, input.Target)
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

func (s *Service) Pack(input PackRequest) (PackResponse, error) {
	return s.PackContext(context.Background(), input)
}

func (s *Service) PackContext(ctx context.Context, input PackRequest) (PackResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PackResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return PackResponse{}, err
	}
	ctx, unlock, err := s.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return PackResponse{}, err
	}
	defer unlock()

	workspacePath, cleanup, err := s.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return PackResponse{}, err
	}
	defer cleanup()

	result, err := s.bundleRules.validate(workspacePath, input.Target)
	if err != nil {
		return PackResponse{}, err
	}
	if len(result.Errors) > 0 {
		return PackResponse{
			WorkspacePath:  input.WorkspacePath,
			PackageName:    input.Target.PackageName,
			Version:        input.Target.Version,
			Warnings:       result.Warnings,
			WarningDetails: warningDetails(result.Warnings),
			ImportReport:   readImportReport(workspacePath, result.Warnings),
			Errors:         result.Errors,
		}, nil
	}

	bundlePath, err := s.workspaceSource.bundlePath(workspacePath, input.Target)
	if s.workspaceStore != nil {
		bundlePath = s.managedBundlePath(input.WorkspacePath, input.Target)
		if err := s.validateManagedBundlePath(input.WorkspacePath, input.Target, bundlePath); err != nil {
			return PackResponse{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
		}
	} else if err != nil {
		return PackResponse{}, err
	}
	bundleBase64 := ""
	if s.workspaceStore != nil {
		bundleBase64, err = s.buildManagedBundle(workspacePath, input.WorkspacePath, input.Target, input.IncludeBundle)
	} else {
		bundleBase64, err = s.bundleRules.buildBundle(workspacePath, bundlePath, input.IncludeBundle)
	}
	if err != nil {
		return PackResponse{}, err
	}
	if s.workspaceStore != nil {
		if saveErr := s.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundlePath, time.Now().UTC()); saveErr != nil {
			return PackResponse{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
		}
		bundlePath = workspaceBundleHandle(input.WorkspacePath, input.Target)
	}

	return PackResponse{
		WorkspacePath:  input.WorkspacePath,
		BundlePath:     bundlePath,
		BundleBase64:   bundleBase64,
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

func (s *Service) Cleanup(workspacePath string) error {
	return s.CleanupContext(context.Background(), workspacePath)
}

func (s *Service) CleanupContext(ctx context.Context, workspacePath string) error {
	if err := packagingContextError(ctx); err != nil {
		return err
	}
	ctx, unlock, err := s.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return err
	}
	defer unlock()

	if s.workspaceStore != nil {
		record, err := s.workspaceStore.LoadWorkspace(ctx, workspacePath)
		if err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		managedRoot, err := s.managedWorkspaceRootFromPath(workspacePath, record.WorkspacePath)
		if err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		if err := os.RemoveAll(managedRoot); err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		if err := s.workspaceStore.DeleteWorkspace(ctx, workspacePath); err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		return nil
	}
	return s.workspaceSource.cleanup(workspacePath)
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
	ctx, unlock, err := s.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return "", err
	}
	defer unlock()

	if s.workspaceStore != nil {
		bundlePath, err := s.workspaceStore.LoadBundle(ctx, workspacePath)
		if err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return "", catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			if err == persist.ErrWorkspaceBundleNotFound {
				return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
			}
			return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		if err := s.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
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
	return s.workspaceSource.bundlePath(workspacePath, target)
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
	ctx, unlock, err := s.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	if s.workspaceStore != nil {
		bundlePath, err := s.workspaceStore.LoadBundle(ctx, workspacePath)
		if err == nil {
			if err := s.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
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
		bundlePath, err = s.workspaceStore.LoadBundle(ctx, workspacePath)
		if err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		if err := s.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		bundleContent, readErr := os.ReadFile(bundlePath)
		if readErr != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		return bundleContent, nil
	}
	bundlePath, err := s.workspaceSource.bundlePath(workspacePath, target)
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
	ctx, unlock, err := s.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	materializedPath, cleanup, err := s.materializeWorkspace(ctx, workspacePath)
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
	if s.releaseRecords == nil || s.publisher == nil {
		return PublishResponse{}, catalog.NewError(500, "internal_error", "Publish workflow is not configured.", nil)
	}
	ctx, unlock, err := s.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return PublishResponse{}, err
	}
	defer unlock()

	packResponse, err := s.PackContext(ctx, PackRequest{
		WorkspacePath: input.WorkspacePath,
		Target:        input.Target,
	})
	if err != nil {
		return PublishResponse{}, err
	}
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
	releaseResult, err := s.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if err == nil {
		return publishResponseFromPack(packResponse, releaseResult, false), nil
	}

	releaseErr := packagingCatalogError(err)
	if releaseErr.Code != "registry_version_not_found" {
		return PublishResponse{}, releaseErr
	}

	packageMetadata, err := s.readWorkspacePackageMetadata(ctx, input.WorkspacePath)
	if err != nil {
		return PublishResponse{}, err
	}
	tarball, err := s.BundleContentContext(ctx, input.WorkspacePath, input.Target)
	if err != nil {
		return PublishResponse{}, err
	}

	publishErr := s.publisher.PublishPackage(ctx, artifactadapter.NPMPublishInput{
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
			releaseResult, err = s.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
			if err == nil {
				return publishResponseFromPack(packResponse, releaseResult, false), nil
			}
		}
		return PublishResponse{}, catalogErr
	}

	releaseResult, err = s.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if err != nil {
		return PublishResponse{}, packagingCatalogError(err)
	}
	return publishResponseFromPack(packResponse, releaseResult, true), nil
}

func (s *Service) PruneExpiredWorkspaces(ttl time.Duration) error {
	if s.workspaceStore != nil && ttl > 0 {
		cutoff := time.Now().UTC().Add(-ttl)
		expiredRecords, err := s.workspaceStore.ListExpired(context.Background(), cutoff)
		if err != nil {
			return err
		}
		for _, record := range expiredRecords {
			lockCtx, unlock, lockErr := s.lockWorkspace(context.Background(), record.WorkspaceID)
			if lockErr != nil {
				return lockErr
			}
			current, loadErr := s.workspaceStore.LoadWorkspace(lockCtx, record.WorkspaceID)
			if loadErr == nil {
				if current.UpdatedAt.Before(cutoff) {
					managedRoot, rootErr := s.managedWorkspaceRootFromPath(current.WorkspaceID, current.WorkspacePath)
					if rootErr != nil {
						unlock()
						return rootErr
					}
					if err := os.RemoveAll(managedRoot); err != nil {
						unlock()
						return err
					}
					if err := s.workspaceStore.DeleteWorkspace(lockCtx, record.WorkspaceID); err != nil {
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
	return s.workspaceSource.pruneExpiredWorkspaces(ttl)
}

func (s *Service) lockWorkspace(ctx context.Context, workspaceID string) (context.Context, func(), error) {
	if s.workspaceStore == nil || strings.TrimSpace(workspaceID) == "" {
		return ctx, func() {}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if current, ok := ctx.Value(workspaceLockContextKey{}).(string); ok && current == workspaceID {
		return ctx, func() {}, nil
	}

	s.workspaceLocksMu.Lock()
	entry := s.workspaceLocks[workspaceID]
	if entry == nil {
		entry = &workspaceLockEntry{}
		s.workspaceLocks[workspaceID] = entry
	}
	entry.refs++
	s.workspaceLocksMu.Unlock()

	entry.mu.Lock()

	release := func() {
		entry.mu.Unlock()

		s.workspaceLocksMu.Lock()
		defer s.workspaceLocksMu.Unlock()

		entry.refs--
		if entry.refs == 0 && s.workspaceLocks[workspaceID] == entry {
			delete(s.workspaceLocks, workspaceID)
		}
	}

	return context.WithValue(ctx, workspaceLockContextKey{}, workspaceID), release, nil
}

func (s *Service) buildManagedBundle(workspacePath string, workspaceID string, target Target, includeBundle bool) (string, error) {
	finalPath := s.managedBundlePath(workspaceID, target)
	if err := s.validateManagedBundlePath(workspaceID, target, finalPath); err != nil {
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

	bundleBase64, err := s.bundleRules.buildBundle(workspacePath, tempPath, includeBundle)
	if err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
	}
	return bundleBase64, nil
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

func (s *Service) readWorkspacePackageMetadata(ctx context.Context, workspacePath string) (workspacePackageMetadata, error) {
	rawMetadata, err := s.PackageMetadataContext(ctx, workspacePath)
	if err != nil {
		return workspacePackageMetadata{}, err
	}
	content, err := json.Marshal(rawMetadata)
	if err != nil {
		return workspacePackageMetadata{}, catalog.NewError(500, "internal_error", "workspace package metadata could not be decoded.", nil)
	}

	var payload workspacePackageMetadata
	if err := json.Unmarshal(content, &payload); err != nil {
		return workspacePackageMetadata{}, catalog.NewError(500, "internal_error", "workspace package metadata could not be decoded.", nil)
	}
	return payload, nil
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
