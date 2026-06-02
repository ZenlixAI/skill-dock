package catalog

import (
	"context"
)

type ArtifactSource interface {
	FindReleaseArtifact(ctx context.Context, packageName string, version string) (ReleaseArtifact, error)
}

type StaticArtifactSource struct {
	artifacts map[string]ReleaseArtifact
}

func NewStaticArtifactSource(artifacts []ReleaseArtifact) *StaticArtifactSource {
	index := make(map[string]ReleaseArtifact, len(artifacts))
	for _, artifact := range artifacts {
		index[releaseArtifactKey(artifact.Version.PackageName, artifact.Version.Version)] = artifact
	}
	return &StaticArtifactSource{artifacts: index}
}

func (s *StaticArtifactSource) FindReleaseArtifact(_ context.Context, packageName string, version string) (ReleaseArtifact, error) {
	artifact, ok := s.artifacts[releaseArtifactKey(packageName, version)]
	if !ok {
		return ReleaseArtifact{}, NewError(404, "registry_version_not_found", "Registry version not found.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}
	return artifact, nil
}
