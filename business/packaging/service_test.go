package packaging

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"skill_registry/business/catalog"
)

func TestValidateTargetAcceptsCompanySpecificSkillScope(t *testing.T) {
	err := validateTarget(Target{PackageName: "@zhenlikeji.skill/zoom-out", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("expected company-specific .skill scope to be valid, got %v", err)
	}
}

func TestValidateTargetRejectsNonSkillScope(t *testing.T) {
	err := validateTarget(Target{PackageName: "@zhenlikeji/zoom-out", Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected non-.skill scope to be rejected")
	}
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		t.Fatalf("expected catalog error, got %T", err)
	}
	if catalogErr.Message != "packageName must use a scoped @<company>.skill/<skill-name> package name." {
		t.Fatalf("unexpected error message: %q", catalogErr.Message)
	}
}

func TestPrepareNormalizesSkillMdOnlyLocalSkillWithoutPackageJSON(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := copyExternalSkillFixture(t, "minimal-imported-skill")

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	wantWarnings := []string{"normalized_missing_package_json"}
	if !reflect.DeepEqual(wantWarnings, resp.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, resp.Warnings)
	}
	if resp.ImportReport == nil || !resp.ImportReport.Normalized || !resp.ImportReport.GeneratedPackageJSON || resp.ImportReport.GeneratedCapabilityPlaceholder {
		t.Fatalf("unexpected import report: %#v", resp.ImportReport)
	}
	if _, err := os.Stat(filepath.Join(resp.WorkspacePath, "docs", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no generated placeholder docs README, got err=%v", err)
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if !validateResp.OK {
		t.Fatalf("expected validate to succeed for SKILL.md-only skill, got errors: %#v", validateResp.Errors)
	}
	if validateResp.ImportReport == nil || !validateResp.ImportReport.GeneratedPackageJSON || validateResp.ImportReport.GeneratedCapabilityPlaceholder {
		t.Fatalf("expected validate import report to preserve normalization details, got %#v", validateResp.ImportReport)
	}

	normalizedFromImport := readNormalizedFromImportFlag(t, resp.WorkspacePath)
	if !normalizedFromImport {
		t.Fatal("expected package.json skill.normalizedFromImport=true")
	}
}

func TestPrepareNormalizesSkillMdOnlyGitSkillWithoutPackageJSON(t *testing.T) {
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

	wantWarnings := []string{"normalized_missing_package_json"}
	if !reflect.DeepEqual(wantWarnings, resp.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, resp.Warnings)
	}
	if _, err := os.Stat(filepath.Join(resp.WorkspacePath, "docs", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no generated placeholder docs README, got err=%v", err)
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if !validateResp.OK {
		t.Fatalf("expected validate to succeed for SKILL.md-only git skill, got errors: %#v", validateResp.Errors)
	}

	normalizedFromImport := readNormalizedFromImportFlag(t, resp.WorkspacePath)
	if !normalizedFromImport {
		t.Fatal("expected package.json skill.normalizedFromImport=true")
	}
}

func TestPrepareStrictModeAllowsSkillMdOnlyImport(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()
	writeSkillMarkdownFixture(t, sourceDir)

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
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
	if !validateResp.OK {
		t.Fatalf("expected validate to succeed in strict mode for SKILL.md-only skill, got errors: %#v", validateResp.Errors)
	}
}

func TestPrepareStrictModeAllowsImportedLocalSkillWhenCapabilityContentExists(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()
	writeSkillMarkdownFixture(t, sourceDir)
	writeCapabilityFixture(t, sourceDir)

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
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

func TestPrepareWarnsWhenImportedPackageJSONIdentityIsRewritten(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := copyExternalSkillFixture(t, "conflicting-package-json-skill")

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
		Options:  Options{Strict: true},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	wantWarnings := []string{"normalized_rewrote_package_json_identity"}
	if !reflect.DeepEqual(wantWarnings, resp.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, resp.Warnings)
	}
	if len(resp.WarningDetails) != 1 || resp.WarningDetails[0].Code != "normalized_rewrote_package_json_identity" {
		t.Fatalf("unexpected warning details: %#v", resp.WarningDetails)
	}
	if resp.ImportReport == nil || !resp.ImportReport.RewrotePackageIdentity || !resp.ImportReport.Normalized {
		t.Fatalf("unexpected import report: %#v", resp.ImportReport)
	}

	content, err := os.ReadFile(filepath.Join(resp.WorkspacePath, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}
	if !strings.Contains(string(content), `"name": "@platform.skill/zoom-out"`) {
		t.Fatalf("expected package.json name to be rewritten, got %q", string(content))
	}
	if !strings.Contains(string(content), `"version": "1.0.0"`) {
		t.Fatalf("expected package.json version to be rewritten, got %q", string(content))
	}
}

func TestPrepareSupportsImportedPluginOnlySkillWithoutPlaceholderDocs(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := copyExternalSkillFixture(t, "plugin-only-skill")

	resp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/plugin-only", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Plugin Only", Description: "Plugin-only imported skill", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	wantWarnings := []string{"normalized_missing_package_json"}
	if !reflect.DeepEqual(wantWarnings, resp.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, resp.Warnings)
	}
	if _, err := os.Stat(filepath.Join(resp.WorkspacePath, "docs", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no placeholder docs README, got err=%v", err)
	}
	if resp.ImportReport == nil || !resp.ImportReport.GeneratedPackageJSON || resp.ImportReport.GeneratedCapabilityPlaceholder {
		t.Fatalf("unexpected import report: %#v", resp.ImportReport)
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/plugin-only", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if !validateResp.OK {
		t.Fatalf("expected validate to succeed, got errors: %#v", validateResp.Errors)
	}
}

func TestPackCreatesBundleFromPlatformAuthoringSource(t *testing.T) {
	service := NewService(t.TempDir())

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md":        "# Capability\n",
				"commands/review.md":    "Use this command to review code.\n",
				"plugin.json":           "{\n  \"name\": \"review-helper\"\n}\n",
				importReportPath:        "{\"should\":\"be ignored in bundle walk\"}\n",
				"agents/codex/hint.txt": "codex-specific hint\n",
			},
		},
		Target: Target{PackageName: "@platform.skill/review-helper", Version: "1.2.3"},
		Metadata: Metadata{
			DisplayName: "review-helper",
			Description: "Review helper skill",
			SourceType:  "platform_authoring",
		},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	packResp, err := service.Pack(PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/review-helper", Version: "1.2.3"},
		IncludeBundle: true,
	})
	if err != nil {
		t.Fatalf("pack returned error: %v", err)
	}
	if len(packResp.Errors) != 0 {
		t.Fatalf("expected pack to succeed, got errors: %#v", packResp.Errors)
	}
	if packResp.BundlePath == "" {
		t.Fatal("expected pack to return a bundle path")
	}
	if packResp.BundleBase64 == "" {
		t.Fatal("expected pack to include bundle base64 when requested")
	}

	bundleBytes, err := base64.StdEncoding.DecodeString(packResp.BundleBase64)
	if err != nil {
		t.Fatalf("decode bundle base64: %v", err)
	}
	archiveEntries := readTarGzEntries(t, bundleBytes)

	for _, required := range []string{
		"package/SKILL.md",
		"package/package.json",
		"package/docs/README.md",
		"package/commands/review.md",
		"package/plugin.json",
		"package/agents/codex/hint.txt",
	} {
		if _, ok := archiveEntries[required]; !ok {
			t.Fatalf("expected bundle to contain %s, got keys=%v", required, sortedMapKeys(archiveEntries))
		}
	}
	if _, ok := archiveEntries["package/"+importReportPath]; ok {
		t.Fatalf("expected bundle to omit %s", importReportPath)
	}
	if !strings.Contains(string(archiveEntries["package/SKILL.md"]), "review-helper") {
		t.Fatalf("expected generated SKILL.md to include display name, got %q", string(archiveEntries["package/SKILL.md"]))
	}
	if !strings.Contains(string(archiveEntries["package/package.json"]), `"name": "@platform.skill/review-helper"`) {
		t.Fatalf("expected package.json to contain package name, got %q", string(archiveEntries["package/package.json"]))
	}
	if !reflect.DeepEqual(packResp.Components, []string{"agents", "commands", "docs", "plugin"}) {
		t.Fatalf("unexpected components (-want +got):\nwant: %#v\ngot:  %#v", []string{"agents", "commands", "docs", "plugin"}, packResp.Components)
	}
}

func TestPackAllowsSkillMdOnlyBundleCreation(t *testing.T) {
	service := NewService(t.TempDir())
	sourceDir := t.TempDir()
	writeSkillMarkdownFixture(t, sourceDir)

	prepareResp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
		Options:  Options{Strict: true},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	packResp, err := service.Pack(PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		IncludeBundle: true,
	})
	if err != nil {
		t.Fatalf("pack returned error: %v", err)
	}
	if len(packResp.Errors) != 0 {
		t.Fatalf("expected pack to succeed for SKILL.md-only skill, got errors: %#v", packResp.Errors)
	}
	if packResp.BundlePath == "" {
		t.Fatal("expected bundle path for SKILL.md-only pack")
	}
	if packResp.BundleBase64 == "" {
		t.Fatal("expected bundle payload for SKILL.md-only pack")
	}
}

func TestPackWritesBundleInsideWorkspace(t *testing.T) {
	service := NewService(t.TempDir())

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/review-helper", Version: "1.2.3"},
		Metadata: Metadata{DisplayName: "review-helper", Description: "Review helper skill", SourceType: "platform_authoring"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	packResp, err := service.Pack(PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/review-helper", Version: "1.2.3"},
	})
	if err != nil {
		t.Fatalf("pack returned error: %v", err)
	}

	relativeBundlePath, err := filepath.Rel(prepareResp.WorkspacePath, packResp.BundlePath)
	if err != nil {
		t.Fatalf("bundle relative path: %v", err)
	}
	if strings.HasPrefix(relativeBundlePath, "..") {
		t.Fatalf("expected bundle path %q to live under workspace %q", packResp.BundlePath, prepareResp.WorkspacePath)
	}
}

func TestCleanupRemovesWorkspaceAndBundle(t *testing.T) {
	service := NewService(t.TempDir())

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/review-helper", Version: "1.2.3"},
		Metadata: Metadata{DisplayName: "review-helper", Description: "Review helper skill", SourceType: "platform_authoring"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	packResp, err := service.Pack(PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/review-helper", Version: "1.2.3"},
	})
	if err != nil {
		t.Fatalf("pack returned error: %v", err)
	}

	if err := service.Cleanup(prepareResp.WorkspacePath); err != nil {
		t.Fatalf("cleanup returned error: %v", err)
	}

	if _, err := os.Stat(prepareResp.WorkspacePath); !os.IsNotExist(err) {
		t.Fatalf("expected workspace to be removed, got err=%v", err)
	}
	if _, err := os.Stat(packResp.BundlePath); !os.IsNotExist(err) {
		t.Fatalf("expected bundle to be removed, got err=%v", err)
	}
}

func TestCleanupExpiredWorkspaces(t *testing.T) {
	tempRoot := t.TempDir()
	service := NewService(tempRoot)

	expiredWorkspace := filepath.Join(tempRoot, "skill-packaging-expired")
	freshWorkspace := filepath.Join(tempRoot, "skill-packaging-fresh")
	otherDirectory := filepath.Join(tempRoot, "other-directory")
	for _, dir := range []string{expiredWorkspace, freshWorkspace, otherDirectory} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	oldTimestamp := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(expiredWorkspace, oldTimestamp, oldTimestamp); err != nil {
		t.Fatalf("chtimes expired workspace: %v", err)
	}

	if err := service.PruneExpiredWorkspaces(1 * time.Hour); err != nil {
		t.Fatalf("prune expired workspaces: %v", err)
	}

	if _, err := os.Stat(expiredWorkspace); !os.IsNotExist(err) {
		t.Fatalf("expected expired workspace to be removed, got err=%v", err)
	}
	if _, err := os.Stat(freshWorkspace); err != nil {
		t.Fatalf("expected fresh workspace to remain, got err=%v", err)
	}
	if _, err := os.Stat(otherDirectory); err != nil {
		t.Fatalf("expected non-packaging directory to remain, got err=%v", err)
	}
}

func TestPrepareRejectsUnsafeInlineFilePath(t *testing.T) {
	service := NewService(t.TempDir())

	_, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"../escape.txt": "nope",
			},
		},
		Target:   Target{PackageName: "@platform.skill/review-helper", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "review-helper", Description: "Review helper"},
	})
	if err == nil {
		t.Fatal("expected prepare to reject unsafe inline file paths")
	}
	catalogErr := err.(*catalog.Error)
	if catalogErr.Code != "unsafe_path_entry" {
		t.Fatalf("expected unsafe_path_entry, got %#v", catalogErr)
	}
}

func TestPackagingRequestGettersExposeIdempotencyAndTarget(t *testing.T) {
	target := Target{PackageName: "@platform.skill/review-helper", Version: "1.0.0"}

	prepare := PrepareRequest{IdempotencyKey: "prepare-key", Target: target}
	if got := prepare.GetIdempotencyKey(); got != "prepare-key" {
		t.Fatalf("prepare GetIdempotencyKey: want %q, got %q", "prepare-key", got)
	}
	if got := prepare.GetTarget(); !reflect.DeepEqual(got, target) {
		t.Fatalf("prepare GetTarget: want %#v, got %#v", target, got)
	}

	validate := ValidateRequest{IdempotencyKey: "validate-key", Target: target}
	if got := validate.GetIdempotencyKey(); got != "validate-key" {
		t.Fatalf("validate GetIdempotencyKey: want %q, got %q", "validate-key", got)
	}
	if got := validate.GetTarget(); !reflect.DeepEqual(got, target) {
		t.Fatalf("validate GetTarget: want %#v, got %#v", target, got)
	}

	pack := PackRequest{IdempotencyKey: "pack-key", Target: target}
	if got := pack.GetIdempotencyKey(); got != "pack-key" {
		t.Fatalf("pack GetIdempotencyKey: want %q, got %q", "pack-key", got)
	}
	if got := pack.GetTarget(); !reflect.DeepEqual(got, target) {
		t.Fatalf("pack GetTarget: want %#v, got %#v", target, got)
	}
}

func TestCopySourceDirectorySkipsPackagingArtifacts(t *testing.T) {
	sourceDir := t.TempDir()
	workspaceDir := t.TempDir()

	writeSkillMarkdownFixture(t, sourceDir)
	if err := os.MkdirAll(filepath.Join(sourceDir, ".dist"), 0o755); err != nil {
		t.Fatalf("mkdir .dist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, ".dist", "bundle.tgz"), []byte("artifact"), 0o644); err != nil {
		t.Fatalf("write .dist artifact: %v", err)
	}

	if err := copySourceDirectory(sourceDir, workspaceDir); err != nil {
		t.Fatalf("copySourceDirectory: %v", err)
	}

	if _, err := os.Stat(filepath.Join(workspaceDir, "SKILL.md")); err != nil {
		t.Fatalf("expected SKILL.md to be copied, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, ".dist")); !os.IsNotExist(err) {
		t.Fatalf("expected .dist to be skipped, got err=%v", err)
	}
}

func TestCreateBundleArchiveSkipsPackagingArtifacts(t *testing.T) {
	workspaceDir := t.TempDir()
	writeSkillMarkdownFixture(t, workspaceDir)
	if err := os.MkdirAll(filepath.Join(workspaceDir, ".dist"), 0o755); err != nil {
		t.Fatalf("mkdir .dist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, ".dist", "bundle.tgz"), []byte("artifact"), 0o644); err != nil {
		t.Fatalf("write .dist artifact: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "bundle.tgz")
	if err := createBundleArchive(workspaceDir, archivePath); err != nil {
		t.Fatalf("createBundleArchive: %v", err)
	}

	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read bundle archive: %v", err)
	}
	entries := readTarGzEntries(t, archiveContent)

	if _, ok := entries["package/SKILL.md"]; !ok {
		t.Fatalf("expected SKILL.md entry, got keys %v", sortedMapKeys(entries))
	}
	if _, ok := entries["package/.dist/bundle.tgz"]; ok {
		t.Fatalf("expected .dist artifact to be excluded, got keys %v", sortedMapKeys(entries))
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

func copyExternalSkillFixture(t *testing.T, fixtureName string) string {
	t.Helper()

	sourceDir := filepath.Join("testdata", "external-skills", fixtureName)
	targetDir := t.TempDir()
	if err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relativePath == "." {
			return nil
		}
		targetPath := filepath.Join(targetDir, relativePath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(targetPath, content, 0o644)
	}); err != nil {
		t.Fatalf("copy fixture %s: %v", fixtureName, err)
	}
	return targetDir
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

func buildSourceArchiveBase64FromDirectory(t *testing.T, root string) string {
	t.Helper()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if shouldIgnorePackagingPath(relativePath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relativePath
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tarWriter.Write(content)
		return err
	})
	if err != nil {
		t.Fatalf("build source archive: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return base64.StdEncoding.EncodeToString(archive.Bytes())
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
			break
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
