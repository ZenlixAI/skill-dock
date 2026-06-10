package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"skill_registry/business/catalog"
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

func TestWorkspaceSourceModuleCreatesMissingTempRootBeforePreparingWorkspace(t *testing.T) {
	tempRoot := filepath.Join(t.TempDir(), "missing", "packaging-workspaces")
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
		t.Fatalf("prepare workspace with missing temp root: %v", err)
	}
	if _, err := os.Stat(tempRoot); err != nil {
		t.Fatalf("expected temp root to be created, got %v", err)
	}
	if !module.managesWorkspace(workspacePath) {
		t.Fatalf("expected managed workspace, got %q", workspacePath)
	}
}

func TestWorkspaceSourceModuleGitCloneUsesShallowClone(t *testing.T) {
	tempRoot := t.TempDir()
	module := newWorkspaceSourceModule(tempRoot, 0*time.Second)

	fakeBin := t.TempDir()
	argsLogPath := filepath.Join(t.TempDir(), "git-args.log")
	gitScript := filepath.Join(fakeBin, "git")
	if err := os.WriteFile(gitScript, []byte(`#!/bin/sh
set -eu
: "${ARGS_LOG_PATH:?}"
printf '%s\n' "$@" > "$ARGS_LOG_PATH"
last=""
for arg in "$@"; do
  last="$arg"
done
mkdir -p "$last/skills/engineering/zoom-out"
printf '# Zoom Out\n' > "$last/skills/engineering/zoom-out/SKILL.md"
`), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	t.Setenv("ARGS_LOG_PATH", argsLogPath)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	workspacePath, err := module.prepare(PrepareRequest{
		Source: Source{
			Type:     "git",
			Location: "https://github.com/mattpocock/skills?path=skills/engineering/zoom-out",
		},
		Target: Target{
			PackageName: "@company.skill/zoom-out",
			Version:     "1.2.0",
		},
	})
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}

	argsRaw, err := os.ReadFile(argsLogPath)
	if err != nil {
		t.Fatalf("read git args: %v", err)
	}
	argsLog := string(argsRaw)
	if !strings.Contains(argsLog, "--depth\n1\n") {
		t.Fatalf("expected shallow clone args, got %q", argsLog)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "SKILL.md")); err != nil {
		t.Fatalf("expected cloned skill to be copied into workspace, got %v", err)
	}
}

func TestWorkspaceSourceModuleGitCloneTimeoutReturnsCloneFailure(t *testing.T) {
	tempRoot := t.TempDir()
	module := newWorkspaceSourceModule(tempRoot, 0*time.Second)

	fakeBin := t.TempDir()
	gitScript := filepath.Join(fakeBin, "git")
	if err := os.WriteFile(gitScript, []byte(`#!/bin/sh
sleep 0.2
exit 0
`), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SKILL_REGISTRY_GIT_CLONE_TIMEOUT_MS", "50")

	_, err := module.prepare(PrepareRequest{
		Source: Source{
			Type:     "git",
			Location: "https://github.com/mattpocock/skills?path=skills/engineering/zoom-out",
		},
		Target: Target{
			PackageName: "@company.skill/zoom-out",
			Version:     "1.2.0",
		},
	})
	if err == nil {
		t.Fatal("expected git clone timeout to fail")
	}

	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		t.Fatalf("expected catalog error, got %T", err)
	}
	if catalogErr.Message != "Failed to clone git source." {
		t.Fatalf("unexpected error message: %q", catalogErr.Message)
	}
	detail, _ := catalogErr.Details["error"].(string)
	if !strings.Contains(strings.ToLower(detail), "timed out") {
		t.Fatalf("expected timeout detail, got %#v", catalogErr.Details)
	}
}
