package application

import (
	"context"
	"errors"
	"strings"
	"time"

	catalog "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
	persist "skill_registry/internal/platform/persistence"
)

type Service struct {
	workspaceService *WorkspaceService
	importReview     *ImportReviewService
	publishService   *PublishService
}

func NewService(tempRoot string) *Service {
	return NewServiceWithWorkspaceStore(tempRoot, nil)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	workspaceService := NewWorkspaceService(tempRoot, store)
	return &Service{
		workspaceService: workspaceService,
		importReview:     NewImportReviewService(workspaceService),
		publishService:   NewPublishService(workspaceService),
	}
}

func (s *Service) ConfigurePublishWorkflow(repo ReleaseRecordCreator, publisher PackagePublisher) {
	if s.publishService == nil {
		s.publishService = NewPublishService(s.workspaceService)
	}
	s.publishService.Configure(repo, publisher)
}

func (s *Service) Review(input packagingdomain.ImportReviewRequest) (packagingdomain.ImportReviewResponse, error) {
	return s.ReviewContext(context.Background(), input)
}

func (s *Service) ReviewContext(ctx context.Context, input packagingdomain.ImportReviewRequest) (packagingdomain.ImportReviewResponse, error) {
	if s.importReview == nil {
		s.importReview = NewImportReviewService(s.workspaceService)
	}
	return s.importReview.ReviewContext(ctx, input)
}

func (s *Service) Publish(input packagingdomain.PublishRequest) (packagingdomain.PublishResponse, error) {
	return s.PublishContext(context.Background(), input)
}

func (s *Service) PublishContext(ctx context.Context, input packagingdomain.PublishRequest) (packagingdomain.PublishResponse, error) {
	if s.publishService == nil {
		s.publishService = NewPublishService(s.workspaceService)
	}
	return s.publishService.Publish(ctx, input)
}

func (s *Service) PruneExpiredWorkspaces(ttl time.Duration) error {
	return s.workspaceService.PruneExpiredWorkspaces(context.Background(), ttl)
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

func publishResponseFromPack(packResponse packagingdomain.PackResponse, releaseResult catalog.ReleaseRecordResult, publishedToRegistry bool) packagingdomain.PublishResponse {
	return packagingdomain.PublishResponse{
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
		WarningDetails:      append([]packagingdomain.WarningDetail(nil), packResponse.WarningDetails...),
		ImportReport:        packResponse.ImportReport,
		Errors:              append([]string(nil), packResponse.Errors...),
	}
}

func packagingCatalogError(err error) *catalog.Error {
	if err == nil {
		return nil
	}
	var catalogErr *catalog.Error
	if errors.As(err, &catalogErr) {
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
