package packaging

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrepareNormalizesLocalSkillWithoutPackageJSONOrCapabilities(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

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
}

func TestPrepareStrictModeDoesNotNormalizeMissingCapabilities(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "local", Location: sourceDir},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
		Options:  Options{Strict: true},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
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
