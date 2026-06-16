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

type WorkflowService interface {
	PrepareContext(ctx context.Context, input packagingdomain.PrepareRequest) (packagingdomain.PrepareResponse, error)
	ValidateContext(ctx context.Context, input packagingdomain.ValidateRequest) (packagingdomain.ValidateResponse, error)
	PackContext(ctx context.Context, input packagingdomain.PackRequest) (packagingdomain.PackResponse, error)
}

type AccessService interface {
	CleanupContext(ctx context.Context, workspacePath string) error
	BundleFilePathContext(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error)
}

type Service struct {
	workspaceService *WorkspaceService
	publishService   *PublishService
}

func NewService(tempRoot string) *Service {
	return NewServiceWithWorkspaceStore(tempRoot, nil)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	workspaceService := NewWorkspaceService(tempRoot, store)
	return &Service{
		workspaceService: workspaceService,
		publishService:   NewPublishService(workspaceService),
	}
}

func (s *Service) ConfigurePublishWorkflow(repo ReleaseRecordCreator, publisher PackagePublisher) {
	if s.publishService == nil {
		s.publishService = NewPublishService(s.workspaceService)
	}
	s.publishService.Configure(repo, publisher)
}

func (s *Service) Prepare(input packagingdomain.PrepareRequest) (packagingdomain.PrepareResponse, error) {
	return s.PrepareContext(context.Background(), input)
}

func (s *Service) PrepareContext(ctx context.Context, input packagingdomain.PrepareRequest) (packagingdomain.PrepareResponse, error) {
	return s.workspaceService.Prepare(ctx, input)
}

func (s *Service) Validate(input packagingdomain.ValidateRequest) (packagingdomain.ValidateResponse, error) {
	return s.ValidateContext(context.Background(), input)
}

func (s *Service) ValidateContext(ctx context.Context, input packagingdomain.ValidateRequest) (packagingdomain.ValidateResponse, error) {
	return s.workspaceService.Validate(ctx, input)
}

func (s *Service) Pack(input packagingdomain.PackRequest) (packagingdomain.PackResponse, error) {
	return s.PackContext(context.Background(), input)
}

func (s *Service) PackContext(ctx context.Context, input packagingdomain.PackRequest) (packagingdomain.PackResponse, error) {
	return s.workspaceService.Pack(ctx, input)
}

func (s *Service) Cleanup(workspacePath string) error {
	return s.CleanupContext(context.Background(), workspacePath)
}

func (s *Service) CleanupContext(ctx context.Context, workspacePath string) error {
	return s.workspaceService.Cleanup(ctx, workspacePath)
}

func (s *Service) BundlePath(workspacePath string, target packagingdomain.Target) (string, error) {
	return s.BundlePathContext(context.Background(), workspacePath, target)
}

func (s *Service) BundlePathContext(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error) {
	return s.workspaceService.BundlePath(ctx, workspacePath, target)
}

func (s *Service) BundleFilePath(workspacePath string, target packagingdomain.Target) (string, error) {
	return s.BundleFilePathContext(context.Background(), workspacePath, target)
}

func (s *Service) BundleFilePathContext(ctx context.Context, workspacePath string, target packagingdomain.Target) (string, error) {
	return s.workspaceService.BundleFilePath(ctx, workspacePath, target)
}

func (s *Service) BundleContent(workspacePath string, target packagingdomain.Target) ([]byte, error) {
	return s.BundleContentContext(context.Background(), workspacePath, target)
}

func (s *Service) BundleContentContext(ctx context.Context, workspacePath string, target packagingdomain.Target) ([]byte, error) {
	return s.workspaceService.BundleContent(ctx, workspacePath, target)
}

func (s *Service) PackageMetadata(workspacePath string) (map[string]any, error) {
	return s.PackageMetadataContext(context.Background(), workspacePath)
}

func (s *Service) PackageMetadataContext(ctx context.Context, workspacePath string) (map[string]any, error) {
	return s.workspaceService.PackageMetadata(ctx, workspacePath)
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

func ensureManagedWorkspace(workspacePath string, managed bool) error {
	if managed {
		return nil
	}
	return catalog.NewError(422, "invalid_request", "workspacePath must reference a managed packaging workspace.", map[string]any{"workspacePath": workspacePath})
}
