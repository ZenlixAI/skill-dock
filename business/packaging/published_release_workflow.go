package packaging

import (
	"context"

	"skill_registry/business/catalog"
	artifactadapter "skill_registry/foundation/artifact"
)

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
	releaseResult, err := w.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if err == nil {
		return publishResponseFromPack(packResponse, releaseResult, false), nil
	}

	releaseErr := packagingCatalogError(err)
	if releaseErr.Code != "registry_version_not_found" {
		return PublishResponse{}, releaseErr
	}

	publishErr := w.publisher.PublishPackage(ctx, artifactadapter.NPMPublishInput{
		PackageName: input.Target.PackageName,
		Version:     input.Target.Version,
		Description: material.metadata.Description,
		DisplayName: material.metadata.Skill.DisplayName,
		SourceType:  material.metadata.Skill.SourceType,
		Tarball:     material.tarball,
	})
	if publishErr != nil {
		catalogErr := packagingCatalogError(publishErr)
		if catalogErr.Code == "registry_version_conflict" {
			releaseResult, err = w.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
			if err == nil {
				return publishResponseFromPack(packResponse, releaseResult, false), nil
			}
		}
		return PublishResponse{}, catalogErr
	}

	releaseResult, err = w.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if err != nil {
		return PublishResponse{}, packagingCatalogError(err)
	}
	return publishResponseFromPack(packResponse, releaseResult, true), nil
}
