package catalogapi

import (
	"context"
	"encoding/json"
	"net/http"

	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	artifactadapter "skill_registry/foundation/artifact"
)

type workspacePackageMetadata struct {
	Description string `json:"description"`
	Skill       struct {
		DisplayName string             `json:"displayName"`
		SourceType  catalog.SourceType `json:"sourceType"`
	} `json:"skill"`
}

func (s *Server) handlePackagingPublish(writer http.ResponseWriter, request *http.Request) {
	if s.packaging == nil {
		http.NotFound(writer, request)
		return
	}

	var payload packaging.PublishRequest
	if err := decodeJSONBody(request, &payload); err != nil {
		s.recordMutationAudit(request, "packaging.publish", "", "", err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		writeCatalogError(writer, request, err)
		return
	}

	if !s.authorizePackageAccess(writer, request, payload.Target.PackageName) {
		return
	}

	response, status, err := s.publishPackagedWorkspace(request.Context(), payload)
	if err != nil {
		s.recordMutationAudit(request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		writeCatalogError(writer, request, err)
		return
	}

	s.recordMutationAudit(request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, status, "succeeded", map[string]any{
		"publishedToRegistry": response.PublishedToRegistry,
		"created":             response.Created,
		"idempotentReplay":    response.IdempotentReplay,
	})
	writeJSON(writer, status, response)
}

func (s *Server) publishPackagedWorkspace(ctx context.Context, payload packaging.PublishRequest) (packaging.PublishResponse, int, *catalog.Error) {
	packResponse, err := s.packaging.Pack(packaging.PackRequest{
		WorkspacePath: payload.WorkspacePath,
		Target:        payload.Target,
	})
	if err != nil {
		catalogErr := catalogErrorFromError(err)
		return packaging.PublishResponse{}, catalogErr.Status, catalogErr
	}
	if len(packResponse.Errors) > 0 {
		return packaging.PublishResponse{}, http.StatusUnprocessableEntity, catalog.NewError(422, "bundle_validation_failed", "Bundle validation failed before publish.", map[string]any{
			"packageName": payload.Target.PackageName,
			"version":     payload.Target.Version,
			"errors":      packResponse.Errors,
		})
	}

	releaseInput := catalog.ReleaseRecordInput{
		ManifestDigest: stringPtr(packResponse.ManifestDigest),
		Components:     packResponse.Components,
	}
	releaseResult, releaseStatus, releaseErr := s.createReleaseRecord(ctx, payload.Target.PackageName, payload.Target.Version, releaseInput)
	if releaseErr == nil {
		return publishResponseFromPack(packResponse, releaseResult, false), releaseStatus, nil
	}
	if releaseErr.Code != "registry_version_not_found" {
		return packaging.PublishResponse{}, releaseErr.Status, releaseErr
	}

	packedMetadata, metadataErr := s.packaging.PackageMetadata(payload.WorkspacePath)
	if metadataErr != nil {
		return packaging.PublishResponse{}, catalogErrorFromError(metadataErr).Status, catalogErrorFromError(metadataErr)
	}
	var packageMetadata workspacePackageMetadata
	rawMetadata, _ := json.Marshal(packedMetadata)
	if err := json.Unmarshal(rawMetadata, &packageMetadata); err != nil {
		return packaging.PublishResponse{}, http.StatusInternalServerError, catalog.NewError(500, "internal_error", "workspace package metadata could not be decoded.", nil)
	}

	tarball, readErr := s.packaging.BundleContent(payload.WorkspacePath, payload.Target)
	if readErr != nil {
		return packaging.PublishResponse{}, http.StatusInternalServerError, catalogErrorFromError(readErr)
	}

	publisher := artifactadapter.NewNPMPublisher(
		s.npmRegistryURL,
		s.npmRegistryPublishAuthorization,
		s.npmRegistryPublishClient,
	)
	publishErr := publisher.PublishPackage(ctx, artifactadapter.NPMPublishInput{
		PackageName: payload.Target.PackageName,
		Version:     payload.Target.Version,
		Description: packageMetadata.Description,
		DisplayName: packageMetadata.Skill.DisplayName,
		SourceType:  packageMetadata.Skill.SourceType,
		Tarball:     tarball,
	})
	if publishErr != nil {
		catalogErr := catalogErrorFromError(publishErr)
		if catalogErr.Code == "registry_version_conflict" {
			releaseResult, releaseStatus, releaseErr = s.createReleaseRecord(ctx, payload.Target.PackageName, payload.Target.Version, releaseInput)
			if releaseErr == nil {
				return publishResponseFromPack(packResponse, releaseResult, false), releaseStatus, nil
			}
		}
		return packaging.PublishResponse{}, catalogErr.Status, catalogErr
	}

	releaseResult, releaseStatus, releaseErr = s.createReleaseRecord(ctx, payload.Target.PackageName, payload.Target.Version, releaseInput)
	if releaseErr != nil {
		return packaging.PublishResponse{}, releaseErr.Status, releaseErr
	}
	return publishResponseFromPack(packResponse, releaseResult, true), releaseStatus, nil
}

func (s *Server) createReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, int, *catalog.Error) {
	result, err := s.repo.CreateReleaseRecord(ctx, packageName, version, input)
	if err != nil {
		catalogErr := catalogErrorFromError(err)
		return catalog.ReleaseRecordResult{}, catalogErr.Status, catalogErr
	}

	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	return result, status, nil
}

func publishResponseFromPack(packResponse packaging.PackResponse, releaseResult catalog.ReleaseRecordResult, publishedToRegistry bool) packaging.PublishResponse {
	return packaging.PublishResponse{
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
		WarningDetails:      append([]packaging.WarningDetail(nil), packResponse.WarningDetails...),
		ImportReport:        packResponse.ImportReport,
		Errors:              append([]string(nil), packResponse.Errors...),
	}
}
