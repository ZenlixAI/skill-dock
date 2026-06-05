package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceSourceModulePreparesPlatformAuthoringWorkspace(t *testing.T) {
	tempRoot := t.TempDir()
	module := newWorkspaceSourceModule(tempRoot, 0*time.Second)

	workspacePath, err := module.prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Metadata: Metadata{
			DisplayName: "review-helper",
			Description: "Review helper skill",
		},
	})
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}
	if !module.managesWorkspace(workspacePath) {
		t.Fatalf("expected managed workspace, got %q", workspacePath)
	}

	skillContent, err := os.ReadFile(filepath.Join(workspacePath, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if !strings.Contains(string(skillContent), "review-helper") {
		t.Fatalf("expected generated SKILL.md to include display name, got %q", string(skillContent))
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "docs", "README.md")); err != nil {
		t.Fatalf("expected authoring file to be materialized, got %v", err)
	}
}
