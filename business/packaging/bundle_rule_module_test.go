package packaging

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBundleRuleModulePrepareWorkspaceWritesPackageMetadataAndImportReport(t *testing.T) {
	workspacePath := t.TempDir()
	writeSkillMarkdownFixture(t, workspacePath)
	module := newBundleRuleModule()

	outcome, err := module.prepareWorkspace(workspacePath, PrepareRequest{
		Source:   Source{Type: "upload_archive"},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare workspace rules: %v", err)
	}

	wantWarnings := []string{"normalized_missing_package_json"}
	if !reflect.DeepEqual(wantWarnings, outcome.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, outcome.Warnings)
	}
	if outcome.ImportReport == nil || !outcome.ImportReport.GeneratedPackageJSON || !outcome.ImportReport.Normalized {
		t.Fatalf("unexpected import report: %#v", outcome.ImportReport)
	}

	content, err := os.ReadFile(filepath.Join(workspacePath, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}
	if !strings.Contains(string(content), `"name": "@platform.skill/zoom-out"`) {
		t.Fatalf("expected package.json name rewrite, got %q", string(content))
	}
	if !strings.Contains(string(content), `"normalizedFromImport": true`) {
		t.Fatalf("expected package.json to mark normalized import, got %q", string(content))
	}
}

func TestBundleRuleModuleValidatesAndBuildsBundle(t *testing.T) {
	workspacePath := t.TempDir()
	target := Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"}
	writeSkillMarkdownFixture(t, workspacePath)
	writeCapabilityFixture(t, workspacePath)
	if err := writePackageJSON(workspacePath, target, Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"}, Options{}, false); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	module := newBundleRuleModule()
	validation, err := module.validate(workspacePath, target)
	if err != nil {
		t.Fatalf("validate workspace: %v", err)
	}
	if len(validation.Errors) != 0 {
		t.Fatalf("expected validation success, got %#v", validation.Errors)
	}
	if !reflect.DeepEqual(validation.Components, []string{"docs"}) {
		t.Fatalf("unexpected components (-want +got):\nwant: %#v\ngot:  %#v", []string{"docs"}, validation.Components)
	}

	bundlePath := filepath.Join(workspacePath, ".dist", "zoom-out-1.0.0.tgz")
	bundleBase64, err := module.buildBundle(workspacePath, bundlePath, true)
	if err != nil {
		t.Fatalf("build bundle: %v", err)
	}
	if bundleBase64 == "" {
		t.Fatal("expected bundle payload when includeBundle is true")
	}

	bundleBytes, err := base64.StdEncoding.DecodeString(bundleBase64)
	if err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	entries := readTarGzEntries(t, bundleBytes)
	if _, ok := entries["package/SKILL.md"]; !ok {
		t.Fatalf("expected SKILL.md in bundle, got keys=%v", sortedMapKeys(entries))
	}
	if _, ok := entries["package/docs/README.md"]; !ok {
		t.Fatalf("expected docs/README.md in bundle, got keys=%v", sortedMapKeys(entries))
	}
}
