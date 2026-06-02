package packaging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPrepareNormalizesLocalSkillWithoutPackageJSONOrCapabilities(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()
	writeSkillMarkdownFixture(t, sourceDir)

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "local", Location: sourceDir},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	wantWarnings := []string{"normalized_missing_package_json", "normalized_missing_capability_content"}
	if !reflect.DeepEqual(wantWarnings, resp.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, resp.Warnings)
	}
	if _, err := os.Stat(filepath.Join(resp.WorkspacePath, "docs", "README.md")); err != nil {
		t.Fatalf("expected placeholder docs README: %v", err)
	}
	readmeContent, err := os.ReadFile(filepath.Join(resp.WorkspacePath, "docs", "README.md"))
	if err != nil {
		t.Fatalf("read placeholder docs README: %v", err)
	}
	if !strings.Contains(string(readmeContent), "generated during packaging normalization") {
		t.Fatalf("expected placeholder docs README to explain normalization, got %q", string(readmeContent))
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if !validateResp.OK {
		t.Fatalf("expected validate to succeed, got errors: %#v", validateResp.Errors)
	}

	normalizedFromImport := readNormalizedFromImportFlag(t, resp.WorkspacePath)
	if !normalizedFromImport {
		t.Fatal("expected package.json skill.normalizedFromImport=true")
	}
}

func TestPrepareNormalizesGitSkillWithoutPackageJSONOrCapabilities(t *testing.T) {
	service := NewService(t.TempDir())
	repoDir := createGitSkillRepo(t)

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "git", Location: repoDir, Ref: "HEAD"},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	wantWarnings := []string{"normalized_missing_package_json", "normalized_missing_capability_content"}
	if !reflect.DeepEqual(wantWarnings, resp.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, resp.Warnings)
	}
	if _, err := os.Stat(filepath.Join(resp.WorkspacePath, "docs", "README.md")); err != nil {
		t.Fatalf("expected placeholder docs README: %v", err)
	}
	readmeContent, err := os.ReadFile(filepath.Join(resp.WorkspacePath, "docs", "README.md"))
	if err != nil {
		t.Fatalf("read placeholder docs README: %v", err)
	}
	if !strings.Contains(string(readmeContent), "generated during packaging normalization") {
		t.Fatalf("expected placeholder docs README to explain normalization, got %q", string(readmeContent))
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if !validateResp.OK {
		t.Fatalf("expected validate to succeed, got errors: %#v", validateResp.Errors)
	}

	normalizedFromImport := readNormalizedFromImportFlag(t, resp.WorkspacePath)
	if !normalizedFromImport {
		t.Fatal("expected package.json skill.normalizedFromImport=true")
	}
}

func TestPrepareStrictModeRewritesPackageJSONButValidateFailsWithoutCapabilityContent(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()
	writeSkillMarkdownFixture(t, sourceDir)

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "local", Location: sourceDir},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
		Options:  Options{Strict: true},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}
	if len(resp.Warnings) != 0 {
		t.Fatalf("expected no normalization warnings in strict mode, got %#v", resp.Warnings)
	}
	if _, err := os.Stat(filepath.Join(resp.WorkspacePath, "package.json")); err != nil {
		t.Fatalf("expected package.json to be written during prepare: %v", err)
	}
	if readNormalizedFromImportFlag(t, resp.WorkspacePath) {
		t.Fatal("expected package.json skill.normalizedFromImport to remain false in strict mode")
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if validateResp.OK {
		t.Fatal("expected validate to fail in strict mode")
	}
	wantErrors := []string{"missing_capability_content"}
	if !reflect.DeepEqual(wantErrors, validateResp.Errors) {
		t.Fatalf("unexpected validate errors (-want +got):\nwant: %#v\ngot:  %#v", wantErrors, validateResp.Errors)
	}
}

func TestPrepareStrictModeAllowsImportedLocalSkillWhenCapabilityContentExists(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()
	writeSkillMarkdownFixture(t, sourceDir)
	writeCapabilityFixture(t, sourceDir)

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "local", Location: sourceDir},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
		Options:  Options{Strict: true},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}
	if len(resp.Warnings) != 0 {
		t.Fatalf("expected no normalization warnings in strict mode, got %#v", resp.Warnings)
	}
	if readNormalizedFromImportFlag(t, resp.WorkspacePath) {
		t.Fatal("expected package.json skill.normalizedFromImport to remain false when no normalization occurred")
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if !validateResp.OK {
		t.Fatalf("expected validate to succeed, got errors: %#v", validateResp.Errors)
	}
}

func writeSkillMarkdownFixture(t *testing.T, dir string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeCapabilityFixture(t *testing.T, dir string) {
	t.Helper()

	capabilityPath := filepath.Join(dir, "docs", "README.md")
	if err := os.MkdirAll(filepath.Dir(capabilityPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capabilityPath, []byte("# Capability\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func createGitSkillRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	if err := runGitCommand("", "init", repoDir); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGitCommand(repoDir, "config", "user.name", "Codex Test"); err != nil {
		t.Fatalf("git config user.name failed: %v", err)
	}
	if err := runGitCommand(repoDir, "config", "user.email", "codex@example.com"); err != nil {
		t.Fatalf("git config user.email failed: %v", err)
	}

	writeSkillMarkdownFixture(t, repoDir)

	if err := runGitCommand(repoDir, "add", "SKILL.md"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := runGitCommand(repoDir, "commit", "-m", "add skill"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
	return repoDir
}

func readNormalizedFromImportFlag(t *testing.T, workspacePath string) bool {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(workspacePath, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}

	var payload struct {
		Skill struct {
			NormalizedFromImport bool `json:"normalizedFromImport"`
		} `json:"skill"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("unmarshal package.json: %v", err)
	}
	return payload.Skill.NormalizedFromImport
}
