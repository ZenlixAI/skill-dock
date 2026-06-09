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
	state, catalogErr := classifyReleaseRecordError(err)
	return releaseRecordAttempt{state: state, err: catalogErr}
}

func (w *publishedReleaseWorkflow) publishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) packagePublishAttempt {
	err := w.publisher.PublishPackage(ctx, input)
	if err == nil {
		return packagePublishAttempt{state: packagePublishWorkflowStatePublished}
	}
	state, catalogErr := classifyPackagePublishError(err)
	return packagePublishAttempt{state: state, err: catalogErr}
}

func classifyReleaseRecordError(err error) (releaseRecordWorkflowState, *catalog.Error) {
	if errors.Is(err, errReleaseArtifactMissing) {
		return releaseRecordWorkflowStateArtifactMissing, releaseArtifactMissingCatalogError(err)
	}

	catalogErr := packagingCatalogError(err)
	switch catalogErr.Code {
	case "registry_version_not_found":
		return releaseRecordWorkflowStateArtifactMissing, catalogErr
	default:
		return releaseRecordWorkflowStateFailed, catalogErr
	}
}

func classifyPackagePublishError(err error) (packagePublishWorkflowState, *catalog.Error) {
	if errors.Is(err, errPackageArtifactAlreadyExists) {
		return packagePublishWorkflowStateArtifactAlreadyExists, packageArtifactAlreadyExistsCatalogError(err)
	}

	catalogErr := packagingCatalogError(err)
	switch catalogErr.Code {
	case "registry_version_conflict":
		return packagePublishWorkflowStateArtifactAlreadyExists, catalogErr
	default:
		return packagePublishWorkflowStateFailed, catalogErr
	}
}

func releaseArtifactMissingCatalogError(err error) *catalog.Error {
	catalogErr := packagingCatalogError(err)
	if catalogErr.Code != "internal_error" {
		return catalogErr
	}
	return catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil)
}

func packageArtifactAlreadyExistsCatalogError(err error) *catalog.Error {
	catalogErr := packagingCatalogError(err)
	if catalogErr.Code != "internal_error" {
		return catalogErr
	}
	return catalog.NewError(409, "registry_version_conflict", "Registry already contains this package version or rejected an overwrite.", nil)
}
