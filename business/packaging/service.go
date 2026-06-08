package packaging

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

type Service struct {
	workspaceSource workspaceSourceModule
	bundleRules     bundleRuleModule
	workspaceStore  persist.WorkspaceStore
}

func NewService(tempRoot string) *Service {
	return NewServiceWithWorkspaceStore(tempRoot, nil)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	return &Service{
		workspaceSource: newWorkspaceSourceModule(tempRoot, 24*time.Hour),
		bundleRules:     newBundleRuleModule(),
		workspaceStore:  store,
	}
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
		handle, persistErr := s.persistWorkspaceSnapshot(ctx, workspacePath)
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
	if err != nil {
		return PackResponse{}, err
	}
	bundleBase64, err := s.bundleRules.buildBundle(workspacePath, bundlePath, input.IncludeBundle)
	if err != nil {
		return PackResponse{}, err
	}
	if s.workspaceStore != nil {
		bundleContent, readErr := os.ReadFile(bundlePath)
		if readErr != nil {
			return PackResponse{}, catalog.NewError(500, "internal_error", "Packed bundle could not be read from the materialized workspace.", nil)
		}
		if saveErr := s.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundleContent, time.Now().UTC()); saveErr != nil {
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
	if s.workspaceStore != nil {
		if _, err := s.workspaceStore.LoadWorkspace(ctx, workspacePath); err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
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
	if s.workspaceStore != nil {
		if _, err := s.workspaceStore.LoadBundle(ctx, workspacePath); err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return "", catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			if err == persist.ErrWorkspaceBundleNotFound {
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
	if s.workspaceStore != nil {
		bundle, err := s.workspaceStore.LoadBundle(ctx, workspacePath)
		if err == nil {
			return bundle, nil
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
		bundle, err = s.workspaceStore.LoadBundle(ctx, workspacePath)
		if err != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		return bundle, nil
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

func (s *Service) PruneExpiredWorkspaces(ttl time.Duration) error {
	if s.workspaceStore != nil && ttl > 0 {
		if err := s.workspaceStore.DeleteExpired(context.Background(), time.Now().UTC().Add(-ttl)); err != nil {
			return err
		}
	}
	return s.workspaceSource.pruneExpiredWorkspaces(ttl)
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

func ensureManagedWorkspace(workspacePath string, managed bool) error {
	if managed {
		return nil
	}
	return catalog.NewError(422, "invalid_request", "workspacePath must reference a managed packaging workspace.", map[string]any{"workspacePath": workspacePath})
}
