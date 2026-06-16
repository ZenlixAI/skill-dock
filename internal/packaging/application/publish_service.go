package application

import (
	"context"
	"errors"
	"slices"

	catalog "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
	artifactadapter "skill_registry/internal/platform/artifact"
)

var (
	ErrReleaseArtifactMissing       = errors.New("published release artifact missing")
	ErrPackageArtifactAlreadyExists = errors.New("published release artifact already exists")
)

type ReleaseRecordWorkflowState int

const (
	ReleaseRecordWorkflowStateReady ReleaseRecordWorkflowState = iota
	ReleaseRecordWorkflowStateArtifactMissing
	ReleaseRecordWorkflowStateFailed
)

type PackagePublishWorkflowState int

const (
	PackagePublishWorkflowStatePublished PackagePublishWorkflowState = iota
	PackagePublishWorkflowStateArtifactAlreadyExists
	PackagePublishWorkflowStateFailed
)

type releaseRecordAttempt struct {
	result catalog.ReleaseRecordResult
	state  ReleaseRecordWorkflowState
	err    error
}

type packagePublishAttempt struct {
	state PackagePublishWorkflowState
	err   error
}

type ReleaseRecordCreator interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error)
}

type PackagePublisher interface {
	PublishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) error
}

type PublishService struct {
	runtime        *WorkspaceService
	releaseRecords ReleaseRecordCreator
	publisher      PackagePublisher
}

func NewPublishService(runtime *WorkspaceService) *PublishService {
	return &PublishService{runtime: runtime}
}

func (w *PublishService) Configure(repo ReleaseRecordCreator, publisher PackagePublisher) {
	w.releaseRecords = repo
	w.publisher = publisher
}

func (w *PublishService) Publish(ctx context.Context, input packagingdomain.PublishRequest) (packagingdomain.PublishResponse, error) {
	if err := packagingContextError(ctx); err != nil {
		return packagingdomain.PublishResponse{}, err
	}
	if w == nil || w.runtime == nil || w.releaseRecords == nil || w.publisher == nil {
		return packagingdomain.PublishResponse{}, catalog.NewError(500, "internal_error", "Publish workflow is not configured.", nil)
	}

	material, err := w.runtime.preparePublishedRelease(ctx, input)
	if err != nil {
		return packagingdomain.PublishResponse{}, err
	}
	defer material.cleanup()

	packResponse := material.packResponse
	if len(packResponse.Errors) > 0 {
		return packagingdomain.PublishResponse{}, catalog.NewError(422, "bundle_validation_failed", "Bundle validation failed before publish.", map[string]any{
			"packageName": input.Target.PackageName,
			"version":     input.Target.Version,
			"errors":      packResponse.Errors,
		})
	}
	if err := verifyReviewSummary(input.Review, packResponse); err != nil {
		return packagingdomain.PublishResponse{}, err
	}

	releaseInput := catalog.ReleaseRecordInput{
		ManifestDigest: stringPtr(packResponse.ManifestDigest),
		Components:     packResponse.Components,
	}
	releaseAttempt := w.createReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	switch releaseAttempt.state {
	case ReleaseRecordWorkflowStateReady:
		return publishResponseFromPack(packResponse, releaseAttempt.result, false), nil
	case ReleaseRecordWorkflowStateFailed:
		return packagingdomain.PublishResponse{}, releaseAttempt.err
	}

	publishAttempt := w.publishPackage(ctx, artifactadapter.NPMPublishInput{
		PackageName: input.Target.PackageName,
		Version:     input.Target.Version,
		Description: material.metadata.Description,
		DisplayName: material.metadata.Skill.DisplayName,
		SourceType:  material.metadata.Skill.SourceType,
		TarballPath: material.tarballPath,
	})
	switch publishAttempt.state {
	case PackagePublishWorkflowStateArtifactAlreadyExists:
		releaseAttempt = w.createReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
		if releaseAttempt.state == ReleaseRecordWorkflowStateReady {
			return publishResponseFromPack(packResponse, releaseAttempt.result, false), nil
		}
		return packagingdomain.PublishResponse{}, releaseAttempt.err
	case PackagePublishWorkflowStateFailed:
		return packagingdomain.PublishResponse{}, publishAttempt.err
	}

	releaseAttempt = w.createReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if releaseAttempt.state != ReleaseRecordWorkflowStateReady {
		return packagingdomain.PublishResponse{}, releaseAttempt.err
	}
	return publishResponseFromPack(packResponse, releaseAttempt.result, true), nil
}

func (w *PublishService) createReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) releaseRecordAttempt {
	result, err := w.releaseRecords.CreateReleaseRecord(ctx, packageName, version, input)
	if err == nil {
		return releaseRecordAttempt{result: result, state: ReleaseRecordWorkflowStateReady}
	}
	state, catalogErr := ClassifyReleaseRecordError(err)
	return releaseRecordAttempt{state: state, err: catalogErr}
}

func (w *PublishService) publishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) packagePublishAttempt {
	err := w.publisher.PublishPackage(ctx, input)
	if err == nil {
		return packagePublishAttempt{state: PackagePublishWorkflowStatePublished}
	}
	state, catalogErr := ClassifyPackagePublishError(err)
	return packagePublishAttempt{state: state, err: catalogErr}
}

func ClassifyReleaseRecordError(err error) (ReleaseRecordWorkflowState, *catalog.Error) {
	if errors.Is(err, ErrReleaseArtifactMissing) {
		return ReleaseRecordWorkflowStateArtifactMissing, ReleaseArtifactMissingCatalogError(err)
	}

	catalogErr := packagingCatalogError(err)
	switch catalogErr.Code {
	case "registry_version_not_found":
		return ReleaseRecordWorkflowStateArtifactMissing, catalogErr
	default:
		return ReleaseRecordWorkflowStateFailed, catalogErr
	}
}

func ClassifyPackagePublishError(err error) (PackagePublishWorkflowState, *catalog.Error) {
	if errors.Is(err, ErrPackageArtifactAlreadyExists) {
		return PackagePublishWorkflowStateArtifactAlreadyExists, PackageArtifactAlreadyExistsCatalogError(err)
	}

	catalogErr := packagingCatalogError(err)
	switch catalogErr.Code {
	case "registry_version_conflict":
		return PackagePublishWorkflowStateArtifactAlreadyExists, catalogErr
	default:
		return PackagePublishWorkflowStateFailed, catalogErr
	}
}

func ReleaseArtifactMissingCatalogError(err error) *catalog.Error {
	catalogErr := packagingCatalogError(err)
	if catalogErr.Code != "internal_error" {
		return catalogErr
	}
	return catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil)
}

func PackageArtifactAlreadyExistsCatalogError(err error) *catalog.Error {
	catalogErr := packagingCatalogError(err)
	if catalogErr.Code != "internal_error" {
		return catalogErr
	}
	return catalog.NewError(409, "registry_version_conflict", "Registry already contains this package version or rejected an overwrite.", nil)
}

func verifyReviewSummary(review packagingdomain.ReviewSummary, packResponse packagingdomain.PackResponse) error {
	if review.ManifestDigest != "" && review.ManifestDigest != packResponse.ManifestDigest {
		return catalog.NewError(409, "review_summary_mismatch", "review manifest digest does not match publish-time package output.", nil)
	}
	if len(review.Components) > 0 && !slices.Equal(review.Components, packResponse.Components) {
		return catalog.NewError(409, "review_summary_mismatch", "review component list does not match publish-time package output.", nil)
	}
	return nil
}
