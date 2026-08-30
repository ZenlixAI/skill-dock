package domain

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestWarningDetailsReturnsStructuredDescriptions(t *testing.T) {
	details := WarningDetails([]string{
		"normalized_missing_package_json",
		"missing_capability_content",
		"normalized_rewrote_package_json_identity",
		"unknown_warning",
	})

	want := []WarningDetail{
		{
			Code:            "normalized_missing_package_json",
			Severity:        "warning",
			Title:           "Generated missing package.json",
			Message:         "The imported skill did not provide a root package.json, so packaging generated one from the requested package_name, version, and metadata.",
			SuggestedAction: "Add and maintain a root package.json in the source skill if you want the package identity to live with the upstream source.",
		},
		{
			Code:            "missing_capability_content",
			Severity:        "high-risk",
			Title:           "Missing capability content",
			Message:         "The imported skill did not contain a root SKILL.md or any recognized capability content such as commands/, agents/, hooks/, scripts/, docs/, plugin.json, or .mcp.json.",
			SuggestedAction: "Add a valid root SKILL.md or real capability content before validate/pack so the bundle contract is satisfied.",
		},
		{
			Code:            "normalized_rewrote_package_json_identity",
			Severity:        "warning",
			Title:           "Rewrote package identity",
			Message:         "The imported skill already had a package.json, but its name and/or version did not match the requested internal target, so packaging rewrote those identity fields.",
			SuggestedAction: "Review whether the internal package_name/version mapping is what you want, and align the upstream package.json if this import should become the long-term source of truth.",
		},
		{
			Code:     "unknown_warning",
			Severity: "warning",
			Title:    "unknown_warning",
			Message:  "unknown_warning",
		},
	}
	if !reflect.DeepEqual(want, details) {
		t.Fatalf("unexpected warning details (-want +got):\nwant: %#v\ngot:  %#v", want, details)
	}
}

func TestBundleRulesPrepareWorkspaceWritesPackageMetadataAndImportReport(t *testing.T) {
	workspacePath := t.TempDir()
	writeSkillMarkdownFixture(t, workspacePath)
	rules := NewBundleRules()

	outcome, err := rules.PrepareWorkspace(workspacePath, PrepareRequest{
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

func TestBundleRulesValidateAndBuildBundle(t *testing.T) {
	workspacePath := t.TempDir()
	target := Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"}
	writeSkillMarkdownFixture(t, workspacePath)
	writeCapabilityFixture(t, workspacePath)
	if err := WritePackageJSON(workspacePath, target, Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"}, Options{}, false); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	rules := NewBundleRules()
	validation, err := rules.Validate(workspacePath, target)
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
	bundleBase64, err := rules.BuildBundle(workspacePath, bundlePath, true)
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

func TestBundleRulesValidateIgnoresEmptyCapabilityDirectories(t *testing.T) {
	workspacePath := t.TempDir()
	target := Target{PackageName: "@platform.skill/empty-docs", Version: "1.0.0"}
	writeSkillMarkdownFixture(t, workspacePath)
	if err := os.MkdirAll(filepath.Join(workspacePath, "docs"), 0o755); err != nil {
		t.Fatalf("create docs directory: %v", err)
	}
	if err := WritePackageJSON(workspacePath, target, Metadata{DisplayName: "Empty Docs", Description: "Broader context", SourceType: "imported"}, Options{}, false); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	rules := NewBundleRules()
	validation, err := rules.Validate(workspacePath, target)
	if err != nil {
		t.Fatalf("validate workspace: %v", err)
	}
	if len(validation.Errors) != 0 {
		t.Fatalf("expected validation success, got %#v", validation.Errors)
	}
	if !reflect.DeepEqual(validation.Components, []string{}) {
		t.Fatalf("unexpected components (-want +got):\nwant: %#v\ngot:  %#v", []string{}, validation.Components)
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

func readTarGzEntries(t *testing.T, archive []byte) map[string][]byte {
	t.Helper()

	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("open gzip archive: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	entries := map[string][]byte{}
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("read tar entry: %v", err)
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read tar entry content: %v", err)
		}
		entries[header.Name] = content
	}
	return entries
}

func sortedMapKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
