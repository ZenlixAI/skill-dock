package packaging

import (
	"context"
	"errors"
	"strings"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
	packagingapplication "skill_registry/internal/packaging/application"
)

type Service struct {
	impl *packagingapplication.Service
}

func NewService(tempRoot string) *Service {
	return NewServiceWithWorkspaceStore(tempRoot, nil)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	return &Service{impl: packagingapplication.NewServiceWithWorkspaceStore(tempRoot, store)}
}

func (s *Service) ConfigurePublishWorkflow(repo releaseRecordCreator, publisher packagePublisher) {
	s.impl.ConfigurePublishWorkflow(repo, publisher)
}

func (s *Service) Prepare(input PrepareRequest) (PrepareResponse, error) {
	return s.PrepareContext(context.Background(), input)
}

func (s *Service) PrepareContext(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	return s.impl.PrepareContext(ctx, input)
}

func (s *Service) Validate(input ValidateRequest) (ValidateResponse, error) {
	return s.ValidateContext(context.Background(), input)
}

func (s *Service) ValidateContext(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	return s.impl.ValidateContext(ctx, input)
}

func (s *Service) Pack(input PackRequest) (PackResponse, error) {
	return s.PackContext(context.Background(), input)
}

func (s *Service) PackContext(ctx context.Context, input PackRequest) (PackResponse, error) {
	return s.impl.PackContext(ctx, input)
}

func (s *Service) Cleanup(workspacePath string) error {
	return s.CleanupContext(context.Background(), workspacePath)
}

func (s *Service) CleanupContext(ctx context.Context, workspacePath string) error {
	return s.impl.CleanupContext(ctx, workspacePath)
}

func (s *Service) BundlePath(workspacePath string, target Target) (string, error) {
	return s.BundlePathContext(context.Background(), workspacePath, target)
}

func (s *Service) BundlePathContext(ctx context.Context, workspacePath string, target Target) (string, error) {
	return s.impl.BundlePathContext(ctx, workspacePath, target)
}

func (s *Service) BundleFilePath(workspacePath string, target Target) (string, error) {
	return s.BundleFilePathContext(context.Background(), workspacePath, target)
}

func (s *Service) BundleFilePathContext(ctx context.Context, workspacePath string, target Target) (string, error) {
	return s.impl.BundleFilePathContext(ctx, workspacePath, target)
}

func (s *Service) BundleContent(workspacePath string, target Target) ([]byte, error) {
	return s.BundleContentContext(context.Background(), workspacePath, target)
}

func (s *Service) BundleContentContext(ctx context.Context, workspacePath string, target Target) ([]byte, error) {
	return s.impl.BundleContentContext(ctx, workspacePath, target)
}

func (s *Service) PackageMetadata(workspacePath string) (map[string]any, error) {
	return s.PackageMetadataContext(context.Background(), workspacePath)
}

func (s *Service) PackageMetadataContext(ctx context.Context, workspacePath string) (map[string]any, error) {
	return s.impl.PackageMetadataContext(ctx, workspacePath)
}

func (s *Service) Publish(input PublishRequest) (PublishResponse, error) {
	return s.PublishContext(context.Background(), input)
}

func (s *Service) PublishContext(ctx context.Context, input PublishRequest) (PublishResponse, error) {
	return s.impl.PublishContext(ctx, input)
}

func (s *Service) PruneExpiredWorkspaces(ttl time.Duration) error {
	return s.impl.PruneExpiredWorkspaces(ttl)
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
