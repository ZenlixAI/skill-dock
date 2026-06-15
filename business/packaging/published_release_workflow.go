package packaging

import (
	"context"

	"skill_registry/business/catalog"
	packagingapplication "skill_registry/internal/packaging/application"
)

var (
	errReleaseArtifactMissing       = packagingapplication.ErrReleaseArtifactMissing
	errPackageArtifactAlreadyExists = packagingapplication.ErrPackageArtifactAlreadyExists
)

type releaseRecordWorkflowState = packagingapplication.ReleaseRecordWorkflowState
type packagePublishWorkflowState = packagingapplication.PackagePublishWorkflowState

const (
	releaseRecordWorkflowStateReady           = packagingapplication.ReleaseRecordWorkflowStateReady
	releaseRecordWorkflowStateArtifactMissing = packagingapplication.ReleaseRecordWorkflowStateArtifactMissing
	releaseRecordWorkflowStateFailed          = packagingapplication.ReleaseRecordWorkflowStateFailed

	packagePublishWorkflowStatePublished             = packagingapplication.PackagePublishWorkflowStatePublished
	packagePublishWorkflowStateArtifactAlreadyExists = packagingapplication.PackagePublishWorkflowStateArtifactAlreadyExists
	packagePublishWorkflowStateFailed                = packagingapplication.PackagePublishWorkflowStateFailed
)

type releaseRecordCreator = packagingapplication.ReleaseRecordCreator
type packagePublisher = packagingapplication.PackagePublisher

type publishedReleaseWorkflow struct {
	impl *packagingapplication.PublishService
}

func newPublishedReleaseWorkflow(runtime *packagingWorkspaceRuntime) *publishedReleaseWorkflow {
	return &publishedReleaseWorkflow{impl: packagingapplication.NewPublishService(runtime.impl)}
}

func (w *publishedReleaseWorkflow) Configure(repo releaseRecordCreator, publisher packagePublisher) {
	w.impl.Configure(repo, publisher)
}

func (w *publishedReleaseWorkflow) Publish(ctx context.Context, input PublishRequest) (PublishResponse, error) {
	return w.impl.Publish(ctx, input)
}

func classifyReleaseRecordError(err error) (releaseRecordWorkflowState, *catalog.Error) {
	return packagingapplication.ClassifyReleaseRecordError(err)
}

func classifyPackagePublishError(err error) (packagePublishWorkflowState, *catalog.Error) {
	return packagingapplication.ClassifyPackagePublishError(err)
}

func releaseArtifactMissingCatalogError(err error) *catalog.Error {
	return packagingapplication.ReleaseArtifactMissingCatalogError(err)
}

func packageArtifactAlreadyExistsCatalogError(err error) *catalog.Error {
	return packagingapplication.PackageArtifactAlreadyExistsCatalogError(err)
}
