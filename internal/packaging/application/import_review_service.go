package application

import (
	"context"

	catalog "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
)

type ReviewService interface {
	ReviewContext(ctx context.Context, input packagingdomain.ImportReviewRequest) (packagingdomain.ImportReviewResponse, error)
}

type ImportReviewService struct {
	workspace *WorkspaceService
}

func NewImportReviewService(workspace *WorkspaceService) *ImportReviewService {
	return &ImportReviewService{workspace: workspace}
}

func (s *ImportReviewService) Review(ctx context.Context, input packagingdomain.ImportReviewRequest) (packagingdomain.ImportReviewResponse, error) {
	return s.ReviewContext(ctx, input)
}

func (s *ImportReviewService) ReviewContext(ctx context.Context, input packagingdomain.ImportReviewRequest) (packagingdomain.ImportReviewResponse, error) {
	if s == nil || s.workspace == nil {
		if err := packagingContextError(ctx); err != nil {
			return packagingdomain.ImportReviewResponse{}, err
		}
		return packagingdomain.ImportReviewResponse{}, catalog.NewError(500, "internal_error", "Import review workflow is not configured.", nil)
	}

	prepared, err := s.workspace.Prepare(ctx, packagingdomain.PrepareRequest{
		IdempotencyKey: input.IdempotencyKey,
		Source:         input.Source,
		Target:         input.Target,
		Metadata:       input.Metadata,
		Options:        input.Options,
	})
	if err != nil {
		return packagingdomain.ImportReviewResponse{}, err
	}
	defer func() {
		_ = s.workspace.Cleanup(ctx, prepared.WorkspacePath)
	}()

	packed, err := s.workspace.Pack(ctx, packagingdomain.PackRequest{
		WorkspacePath: prepared.WorkspacePath,
		Target:        input.Target,
		IncludeBundle: input.IncludeBundleBase64,
	})
	if err != nil {
		return packagingdomain.ImportReviewResponse{}, err
	}

	warnings := mergeWarnings(prepared.Warnings, packed.Warnings)
	return packagingdomain.ImportReviewResponse{
		OK:             len(packed.Errors) == 0,
		PackageName:    packed.PackageName,
		Version:        packed.Version,
		ManifestDigest: packed.ManifestDigest,
		Components:     append([]string(nil), packed.Components...),
		BundleBase64:   packed.BundleBase64,
		Warnings:       warnings,
		WarningDetails: packagingdomain.WarningDetails(warnings),
		ImportReport:   firstImportReport(packed.ImportReport, prepared.ImportReport),
		Errors:         append([]string(nil), packed.Errors...),
	}, nil
}

func mergeWarnings(groups ...[]string) []string {
	if len(groups) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	merged := make([]string, 0)
	for _, group := range groups {
		for _, warning := range group {
			if _, ok := seen[warning]; ok {
				continue
			}
			seen[warning] = struct{}{}
			merged = append(merged, warning)
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func firstImportReport(reports ...*packagingdomain.ImportReport) *packagingdomain.ImportReport {
	for _, report := range reports {
		if report != nil {
			return report
		}
	}
	return nil
}
