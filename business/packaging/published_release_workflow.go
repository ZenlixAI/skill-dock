package packaging

import (
	"context"
	"errors"

	"skill_registry/business/catalog"
	artifactadapter "skill_registry/foundation/artifact"
)

var (
	errReleaseArtifactMissing       = errors.New("published release artifact missing")
	errPackageArtifactAlreadyExists = errors.New("published release artifact already exists")
)

type releaseRecordWorkflowState int

const (
	releaseRecordWorkflowStateReady releaseRecordWorkflowState = iota
	releaseRecordWorkflowStateArtifactMissing
	releaseRecordWorkflowStateFailed
)

type packagePublishWorkflowState int

const (
	packagePublishWorkflowStatePublished packagePublishWorkflowState = iota
	packagePublishWorkflowStateArtifactAlreadyExists
	packagePublishWorkflowStateFailed
)

type releaseRecordAttempt struct {
	result catalog.ReleaseRecordResult
	state  releaseRecordWorkflowState
	err    error
}

type packagePublishAttempt struct {
	state packagePublishWorkflowState
	err   error
}

type releaseRecordCreator interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error)
}

type packagePublisher interface {
	PublishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) error
}

type publishedReleaseWorkflow struct {
	runtime        *packagingWorkspaceRuntime
	releaseRecords releaseRecordCreator
	publisher      packagePublisher
}

func newPublishedReleaseWorkflow(runtime *packagingWorkspaceRuntime) *publishedReleaseWorkflow {
	return &publishedReleaseWorkflow{runtime: runtime}
}

func (w *publishedReleaseWorkflow) Configure(repo releaseRecordCreator, publisher packagePublisher) {
	w.releaseRecords = repo
	w.publisher = publisher
}

func (w *publishedReleaseWorkflow) Publish(ctx context.Context, input PublishRequest) (PublishResponse, error) {
	if err := packagingContextError(ctx); err != nil {
		return PublishResponse{}, err
	}
	if w == nil || w.runtime == nil || w.releaseRecords == nil || w.publisher == nil {
		return PublishResponse{}, catalog.NewError(500, "internal_error", "Publish workflow is not configured.", nil)
	}

	material, err := w.runtime.preparePublishedRelease(ctx, input)
	if err != nil {
		return PublishResponse{}, err
	}
	defer material.cleanup()

	packResponse := material.packResponse
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
	releaseAttempt := w.createReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	switch releaseAttempt.state {
	case releaseRecordWorkflowStateReady:
		return publishResponseFromPack(packResponse, releaseAttempt.result, false), nil
	case releaseRecordWorkflowStateFailed:
		return PublishResponse{}, releaseAttempt.err
	}

	publishAttempt := w.publishPackage(ctx, artifactadapter.NPMPublishInput{
		PackageName: input.Target.PackageName,
		Version:     input.Target.Version,
		Description: material.metadata.Description,
		DisplayName: material.metadata.Skill.DisplayName,
		SourceType:  material.metadata.Skill.SourceType,
		Tarball:     material.tarball,
	})
	switch publishAttempt.state {
	case packagePublishWorkflowStateArtifactAlreadyExists:
		releaseAttempt = w.createReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
		if releaseAttempt.state == releaseRecordWorkflowStateReady {
			return publishResponseFromPack(packResponse, releaseAttempt.result, false), nil
		}
		return PublishResponse{}, publishAttempt.err
	case packagePublishWorkflowStateFailed:
		return PublishResponse{}, publishAttempt.err
	}

	releaseAttempt = w.createReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if releaseAttempt.state != releaseRecordWorkflowStateReady {
		return PublishResponse{}, releaseAttempt.err
	}
	return publishResponseFromPack(packResponse, releaseAttempt.result, true), nil
}

func (w *publishedReleaseWorkflow) createReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) releaseRecordAttempt {
	result, err := w.releaseRecords.CreateReleaseRecord(ctx, packageName, version, input)
	if err == nil {
		return releaseRecordAttempt{result: result, state: releaseRecordWorkflowStateReady}
	}
	catalogErr := packagingCatalogError(err)
	if errors.Is(err, errReleaseArtifactMissing) {
		if catalogErr.Code == "internal_error" {
			catalogErr = catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil)
		}
		return releaseRecordAttempt{state: releaseRecordWorkflowStateArtifactMissing, err: catalogErr}
	}
	if catalogErr.Code == "registry_version_not_found" {
		return releaseRecordAttempt{state: releaseRecordWorkflowStateArtifactMissing, err: catalogErr}
	}
	return releaseRecordAttempt{state: releaseRecordWorkflowStateFailed, err: catalogErr}
}

func (w *publishedReleaseWorkflow) publishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) packagePublishAttempt {
	err := w.publisher.PublishPackage(ctx, input)
	if err == nil {
		return packagePublishAttempt{state: packagePublishWorkflowStatePublished}
	}
	catalogErr := packagingCatalogError(err)
	if errors.Is(err, errPackageArtifactAlreadyExists) {
		if catalogErr.Code == "internal_error" {
			catalogErr = catalog.NewError(409, "registry_version_conflict", "Registry already contains this package version or rejected an overwrite.", nil)
		}
		return packagePublishAttempt{state: packagePublishWorkflowStateArtifactAlreadyExists, err: catalogErr}
	}
	if catalogErr.Code == "registry_version_conflict" {
		return packagePublishAttempt{state: packagePublishWorkflowStateArtifactAlreadyExists, err: catalogErr}
	}
	return packagePublishAttempt{state: packagePublishWorkflowStateFailed, err: catalogErr}
}
