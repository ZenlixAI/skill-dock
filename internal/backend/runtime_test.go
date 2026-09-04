package backend

import (
	"context"
	"path/filepath"
	"testing"

	cataloginfra "skill_registry/internal/catalog/infra"
)

func TestNewRuntimeSeedsRepository(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(context.Background(), Config{
		DatabaseURL:         "file:" + filepath.Join(root, "runtime.db"),
		SharedWorkspaceRoot: filepath.Join(root, "workspaces"),
		ArtifactSource:      cataloginfra.NewStaticArtifactSource(cataloginfra.SampleData().Artifacts),
		RepositoryOptions: cataloginfra.SQLRepositoryOptions{
			DisableSeed: false,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	skill, err := runtime.Repository.GetSkill(context.Background(), "@platform.skill/pdf-skill")
	if err != nil {
		t.Fatalf("expected seeded skill, got error: %v", err)
	}
	if skill.PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("expected seeded skill package, got %q", skill.PackageName)
	}
}
