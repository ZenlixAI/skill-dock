package artifact

import (
	"reflect"
	"testing"

	platformartifact "skill_registry/internal/platform/artifact"
)

func TestPublisherAliasMatchesPlatformType(t *testing.T) {
	if reflect.TypeOf((*NPMPublisher)(nil)) != reflect.TypeOf((*platformartifact.NPMPublisher)(nil)) {
		t.Fatalf("expected NPMPublisher to alias platform NPMPublisher")
	}
	if reflect.TypeOf((*NPMRegistryArtifactSource)(nil)) != reflect.TypeOf((*platformartifact.NPMRegistryArtifactSource)(nil)) {
		t.Fatalf("expected NPMRegistryArtifactSource to alias platform NPMRegistryArtifactSource")
	}
}
