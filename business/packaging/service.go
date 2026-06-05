package packaging

import (
	"os"
	"strings"
	"time"

	"skill_registry/business/catalog"
)

type Service struct {
	workspaceSource workspaceSourceModule
	bundleRules     bundleRuleModule
}

func NewService(tempRoot string) *Service {
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	return &Service{
		workspaceSource: newWorkspaceSourceModule(tempRoot, 24*time.Hour),
		bundleRules:     newBundleRuleModule(),
	}
}

func (s *Service) Prepare(input PrepareRequest) (PrepareResponse, error) {
	if err := validateTarget(input.Target); err != nil {
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
	if err := validateTarget(input.Target); err != nil {
		return ValidateResponse{}, err
	}

	result, err := s.bundleRules.validate(input.WorkspacePath, input.Target)
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
		ImportReport:   readImportReport(input.WorkspacePath, result.Warnings),
		Errors:         result.Errors,
	}, nil
}

func (s *Service) Pack(input PackRequest) (PackResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PackResponse{}, err
	}

	result, err := s.bundleRules.validate(input.WorkspacePath, input.Target)
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
			ImportReport:   readImportReport(input.WorkspacePath, result.Warnings),
			Errors:         result.Errors,
		}, nil
	}

	bundlePath, err := s.workspaceSource.bundlePath(input.WorkspacePath, input.Target)
	if err != nil {
		return PackResponse{}, err
	}
	bundleBase64, err := s.bundleRules.buildBundle(input.WorkspacePath, bundlePath, input.IncludeBundle)
	if err != nil {
		return PackResponse{}, err
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
		ImportReport:   readImportReport(input.WorkspacePath, result.Warnings),
		Errors:         result.Errors,
	}, nil
}

func (s *Service) Cleanup(workspacePath string) error {
	return s.workspaceSource.cleanup(workspacePath)
}

func (s *Service) BundlePath(workspacePath string, target Target) (string, error) {
	if err := validateTarget(target); err != nil {
		return "", err
	}
	return s.workspaceSource.bundlePath(workspacePath, target)
}

func (s *Service) PruneExpiredWorkspaces(ttl time.Duration) error {
	return s.workspaceSource.pruneExpiredWorkspaces(ttl)
}

func ensureManagedWorkspace(workspacePath string, managed bool) error {
	if managed {
		return nil
	}
	return catalog.NewError(422, "invalid_request", "workspacePath must reference a managed packaging workspace.", map[string]any{"workspacePath": workspacePath})
}
