package artifact

import platformartifact "skill_registry/internal/platform/artifact"

type JSONArtifactSource = platformartifact.JSONArtifactSource
type NPMRegistryArtifactSource = platformartifact.NPMRegistryArtifactSource

var (
	NewJSONArtifactSource        = platformartifact.NewJSONArtifactSource
	NewNPMRegistryArtifactSource = platformartifact.NewNPMRegistryArtifactSource
)
