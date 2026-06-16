package infra

import (
	"context"

	catalogdomain "skill_registry/internal/catalog/domain"
)

type StaticArtifactSource struct {
	artifacts map[string]catalogdomain.ReleaseArtifact
}

func NewStaticArtifactSource(artifacts []catalogdomain.ReleaseArtifact) *StaticArtifactSource {
	index := make(map[string]catalogdomain.ReleaseArtifact, len(artifacts))
	for _, artifact := range artifacts {
		index[artifact.Version.PackageName+"@"+artifact.Version.Version] = artifact
	}
	return &StaticArtifactSource{artifacts: index}
}

func (s *StaticArtifactSource) FindReleaseArtifact(_ context.Context, packageName string, version string) (catalogdomain.ReleaseArtifact, error) {
	artifact, ok := s.artifacts[packageName+"@"+version]
	if !ok {
		return catalogdomain.ReleaseArtifact{}, catalogdomain.NewError(404, "registry_version_not_found", "Registry version not found.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}
	return artifact, nil
}
