package packaging

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"skill_registry/business/catalog"
	artifactadapter "skill_registry/foundation/artifact"
	"skill_registry/foundation/persist"
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

func TestPrepareWithWorkspaceStoreMovesWorkspaceIntoManagedRoot(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)

	resp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "managed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	if filepath.IsAbs(resp.WorkspacePath) {
		t.Fatalf("expected workspace handle, got absolute path %q", resp.WorkspacePath)
	}

	managedWorkspace := filepath.Join(root, resp.WorkspacePath, "workspace")
	if _, err := os.Stat(filepath.Join(managedWorkspace, "SKILL.md")); err != nil {
		t.Fatalf("expected managed workspace SKILL.md: %v", err)
	}

	validateResp, err := service.Validate(ValidateRequest{
		WorkspacePath: resp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("validate managed workspace: %v", err)
	}
	if !validateResp.OK {
		t.Fatalf("expected managed workspace validate to succeed, got errors %#v", validateResp.Errors)
	}

	metadata, err := service.PackageMetadata(resp.WorkspacePath)
	if err != nil {
		t.Fatalf("package metadata from managed workspace: %v", err)
	}
	if metadata["name"] != "@platform.skill/managed" {
		t.Fatalf("expected package metadata name, got %#v", metadata["name"])
	}

	record, err := store.LoadWorkspace(context.Background(), resp.WorkspacePath)
	if err != nil {
		t.Fatalf("load stored workspace metadata: %v", err)
	}
	if record.WorkspacePath != managedWorkspace {
		t.Fatalf("expected stored managed workspace path %q, got %q", managedWorkspace, record.WorkspacePath)
	}
}

func TestBundleContentReusesPersistedBundlePath(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "managed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	packResp, err := service.Pack(PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if !strings.HasPrefix(packResp.BundlePath, "workspace://") {
		t.Fatalf("expected external bundle handle, got %q", packResp.BundlePath)
	}

	first, err := service.BundleContent(prepareResp.WorkspacePath, Target{PackageName: "@platform.skill/managed", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("first bundle read: %v", err)
	}
	second, err := service.BundleContent(prepareResp.WorkspacePath, Target{PackageName: "@platform.skill/managed", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("second bundle read: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("expected stable persisted bundle content")
	}

	record, err := store.LoadWorkspace(context.Background(), prepareResp.WorkspacePath)
	if err != nil {
		t.Fatalf("load stored workspace metadata: %v", err)
	}
	if record.BundlePath == "" {
		t.Fatal("expected persisted bundle path")
	}
	if _, err := os.Stat(record.BundlePath); err != nil {
		t.Fatalf("expected persisted bundle file to exist: %v", err)
	}
}

func TestCleanupWithWorkspaceStoreRemovesManagedWorkspaceTree(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "managed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	if _, err := service.Pack(PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
	}); err != nil {
		t.Fatalf("pack: %v", err)
	}

	managedRoot := filepath.Join(root, prepareResp.WorkspacePath)
	if err := service.Cleanup(prepareResp.WorkspacePath); err != nil {
		t.Fatalf("cleanup with workspace store: %v", err)
	}
	if _, err := os.Stat(managedRoot); !os.IsNotExist(err) {
		t.Fatalf("expected managed root to be removed, got err=%v", err)
	}
	if _, err := store.LoadWorkspace(context.Background(), prepareResp.WorkspacePath); !errors.Is(err, persist.ErrWorkspaceRecordNotFound) {
		t.Fatalf("expected store row to be removed, got %v", err)
	}
}

func TestCleanupWithWorkspaceStoreRejectsNonCanonicalManagedWorkspacePath(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)

	poisonRoot := filepath.Join(root, "ws-poison")
	poisonPath := filepath.Join(poisonRoot, "unexpected")
	if err := os.MkdirAll(poisonPath, 0o755); err != nil {
		t.Fatalf("mkdir poison path: %v", err)
	}
	store.records["ws-poison"] = persist.WorkspaceRecord{
		WorkspaceID:   "ws-poison",
		WorkspacePath: poisonPath,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	err := service.Cleanup("ws-poison")
	if err == nil {
		t.Fatal("expected cleanup to reject non-canonical managed path")
	}
	if _, statErr := os.Stat(poisonRoot); statErr != nil {
		t.Fatalf("expected poison root to remain after failed cleanup, got %v", statErr)
	}
}

func TestPruneExpiredWorkspacesWithWorkspaceStoreRemovesManagedWorkspaceTree(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)

	expiredResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/expired", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "expired", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare expired workspace: %v", err)
	}
	freshResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/fresh", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "fresh", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare fresh workspace: %v", err)
	}

	expiredRecord := store.records[expiredResp.WorkspacePath]
	expiredRecord.UpdatedAt = time.Now().UTC().Add(-2 * time.Hour)
	store.records[expiredResp.WorkspacePath] = expiredRecord

	if err := service.PruneExpiredWorkspaces(time.Hour); err != nil {
		t.Fatalf("prune expired managed workspaces: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, expiredResp.WorkspacePath)); !os.IsNotExist(err) {
		t.Fatalf("expected expired managed root to be removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, freshResp.WorkspacePath)); err != nil {
		t.Fatalf("expected fresh managed root to remain, got err=%v", err)
	}
	if _, err := store.LoadWorkspace(context.Background(), expiredResp.WorkspacePath); !errors.Is(err, persist.ErrWorkspaceRecordNotFound) {
		t.Fatalf("expected expired store row to be removed, got %v", err)
	}
}

func TestPruneExpiredWorkspacesWithWorkspaceStoreSkipsWorkspaceRefreshedAfterListing(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)

	resp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/refreshed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "refreshed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare refreshed workspace: %v", err)
	}

	store.mu.Lock()
	record := store.records[resp.WorkspacePath]
	record.UpdatedAt = time.Now().UTC().Add(-2 * time.Hour)
	store.records[resp.WorkspacePath] = record
	store.afterListExpired = func() {
		store.mu.Lock()
		defer store.mu.Unlock()
		refreshed := store.records[resp.WorkspacePath]
		refreshed.UpdatedAt = time.Now().UTC()
		store.records[resp.WorkspacePath] = refreshed
	}
	store.mu.Unlock()

	if err := service.PruneExpiredWorkspaces(time.Hour); err != nil {
		t.Fatalf("prune refreshed managed workspace: %v", err)
	}

	managedRoot := filepath.Join(root, resp.WorkspacePath)
	if _, err := os.Stat(managedRoot); err != nil {
		t.Fatalf("expected refreshed managed root to remain, got %v", err)
	}
	record, err = store.LoadWorkspace(context.Background(), resp.WorkspacePath)
	if err != nil {
		t.Fatalf("expected refreshed store row to remain, got %v", err)
	}
	if !record.UpdatedAt.After(time.Now().UTC().Add(-time.Minute)) {
		t.Fatalf("expected refreshed updated_at to stay fresh, got %s", record.UpdatedAt)
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

func TestPrepareContextPropagatesCancellationToWorkspaceStore(t *testing.T) {
	service := NewServiceWithWorkspaceStore(t.TempDir(), cancelAwareWorkspaceStore{})
	sourceDir := copyExternalSkillFixture(t, "minimal-imported-skill")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.PrepareContext(ctx, PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err == nil {
		t.Fatal("expected canceled prepare to fail")
	}
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		t.Fatalf("expected catalog error, got %T", err)
	}
	if catalogErr.Code != "request_canceled" {
		t.Fatalf("expected request_canceled, got %#v", catalogErr)
	}
}

func TestPackContextPropagatesCancellationToWorkspaceStore(t *testing.T) {
	workspaceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceDir, "SKILL.md"), []byte(skillMarkdownFixture()), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "package.json"), []byte(`{"name":"@platform.skill/zoom-out","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	store := newMemoryWorkspaceStore()
	store.records["ws-canceled"] = persist.WorkspaceRecord{
		WorkspaceID:   "ws-canceled",
		WorkspacePath: workspaceDir,
	}
	service := NewServiceWithWorkspaceStore(t.TempDir(), store)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.PackContext(ctx, PackRequest{
		WorkspacePath: "ws-canceled",
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err == nil {
		t.Fatal("expected canceled pack to fail")
	}
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		t.Fatalf("expected catalog error, got %T", err)
	}
	if catalogErr.Code != "request_canceled" {
		t.Fatalf("expected request_canceled, got %#v", catalogErr)
	}
}

type cancelAwareWorkspaceStore struct{}

func (cancelAwareWorkspaceStore) SaveWorkspace(ctx context.Context, workspaceID string, workspacePath string, createdAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (cancelAwareWorkspaceStore) LoadWorkspace(ctx context.Context, workspaceID string) (persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return persist.WorkspaceRecord{}, err
	}
	return persist.WorkspaceRecord{}, persist.ErrWorkspaceRecordNotFound
}

func (cancelAwareWorkspaceStore) SaveBundle(ctx context.Context, workspaceID string, bundlePath string, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (cancelAwareWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", persist.ErrWorkspaceBundleNotFound
}

func (cancelAwareWorkspaceStore) ListExpired(ctx context.Context, cutoff time.Time) ([]persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (cancelAwareWorkspaceStore) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	return ctx.Err()
}

func (cancelAwareWorkspaceStore) DeleteExpired(ctx context.Context, cutoff time.Time) error {
	return ctx.Err()
}

type memoryWorkspaceStore struct {
	mu               sync.Mutex
	afterListExpired func()
	records          map[string]persist.WorkspaceRecord
}

func newMemoryWorkspaceStore() *memoryWorkspaceStore {
	return &memoryWorkspaceStore{records: map[string]persist.WorkspaceRecord{}}
}

func (s *memoryWorkspaceStore) SaveWorkspace(ctx context.Context, workspaceID string, workspacePath string, createdAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	record := persist.WorkspaceRecord{
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	if existing, ok := s.records[workspaceID]; ok {
		record.CreatedAt = existing.CreatedAt
	}
	s.records[workspaceID] = record
	return nil
}

func (s *memoryWorkspaceStore) LoadWorkspace(ctx context.Context, workspaceID string) (persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return persist.WorkspaceRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[workspaceID]
	if !ok || strings.TrimSpace(record.WorkspacePath) == "" {
		return persist.WorkspaceRecord{}, persist.ErrWorkspaceRecordNotFound
	}
	return record, nil
}

func (s *memoryWorkspaceStore) SaveBundle(ctx context.Context, workspaceID string, bundlePath string, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[workspaceID]
	if !ok || strings.TrimSpace(record.WorkspacePath) == "" {
		return persist.ErrWorkspaceRecordNotFound
	}
	record.BundlePath = bundlePath
	record.UpdatedAt = updatedAt
	s.records[workspaceID] = record
	return nil
}

func (s *memoryWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[workspaceID]
	if !ok || strings.TrimSpace(record.WorkspacePath) == "" {
		return "", persist.ErrWorkspaceRecordNotFound
	}
	if strings.TrimSpace(record.BundlePath) == "" {
		return "", persist.ErrWorkspaceBundleNotFound
	}
	return record.BundlePath, nil
}

func (s *memoryWorkspaceStore) ListExpired(ctx context.Context, cutoff time.Time) ([]persist.WorkspaceRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	records := []persist.WorkspaceRecord{}
	for _, record := range s.records {
		if !record.UpdatedAt.IsZero() && record.UpdatedAt.Before(cutoff) {
			records = append(records, record)
		}
	}
	hook := s.afterListExpired
	s.mu.Unlock()
	if hook != nil {
		hook()
	}
	return records, nil
}

func (s *memoryWorkspaceStore) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.records, workspaceID)
	return nil
}

func (s *memoryWorkspaceStore) DeleteExpired(ctx context.Context, cutoff time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for workspaceID, record := range s.records {
		if !record.UpdatedAt.IsZero() && record.UpdatedAt.Before(cutoff) {
			delete(s.records, workspaceID)
		}
	}
	return nil
}

func skillMarkdownFixture() string {
	return "---\nname: zoom-out\ndescription: Broader context\n---\n"
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

func TestPublishContextPublishesAndCreatesReleaseRecord(t *testing.T) {
	service := NewService(t.TempDir())
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{err: catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil)},
			{result: catalog.ReleaseRecordResult{Created: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{}
	service.ConfigurePublishWorkflow(repo, publisher)

	sourceDir := copyExternalSkillFixture(t, "minimal-imported-skill")
	prepareResp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	response, err := service.PublishContext(context.Background(), PublishRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if !response.PublishedToRegistry || !response.Created || response.IdempotentReplay {
		t.Fatalf("unexpected publish response: %+v", response)
	}
	if repo.calls != 2 {
		t.Fatalf("expected 2 release record attempts, got %d", repo.calls)
	}
	if publisher.calls != 1 {
		t.Fatalf("expected 1 publisher call, got %d", publisher.calls)
	}
	if publisher.lastInput.PackageName != "@platform.skill/zoom-out" || publisher.lastInput.Version != "1.0.0" {
		t.Fatalf("unexpected publisher input: %+v", publisher.lastInput)
	}
}

func TestPublishContextSkipsRegistryPublishWhenReleaseAlreadyExists(t *testing.T) {
	service := NewService(t.TempDir())
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{result: catalog.ReleaseRecordResult{Created: false, IdempotentReplay: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{}
	service.ConfigurePublishWorkflow(repo, publisher)

	sourceDir := copyExternalSkillFixture(t, "minimal-imported-skill")
	prepareResp, err := service.Prepare(PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	response, err := service.PublishContext(context.Background(), PublishRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if response.PublishedToRegistry || response.Created || !response.IdempotentReplay {
		t.Fatalf("unexpected publish response: %+v", response)
	}
	if publisher.calls != 0 {
		t.Fatalf("expected publisher to be skipped, got %d calls", publisher.calls)
	}
}

func TestPublishContextWithWorkspaceStoreBlocksCleanupUntilPublishCompletes(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{err: catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil)},
			{result: catalog.ReleaseRecordResult{Created: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{
		started: make(chan struct{}),
		block:   make(chan struct{}),
	}
	service.ConfigurePublishWorkflow(repo, publisher)

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "managed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	publishDone := make(chan error, 1)
	go func() {
		_, publishErr := service.PublishContext(context.Background(), PublishRequest{
			WorkspacePath: prepareResp.WorkspacePath,
			Target:        Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
		})
		publishDone <- publishErr
	}()

	select {
	case <-publisher.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for publisher to start")
	}

	cleanupDone := make(chan error, 1)
	go func() {
		cleanupDone <- service.Cleanup(prepareResp.WorkspacePath)
	}()

	select {
	case err := <-cleanupDone:
		t.Fatalf("expected cleanup to wait for publish completion, got %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	if _, err := os.Stat(filepath.Join(root, prepareResp.WorkspacePath)); err != nil {
		t.Fatalf("expected managed root to remain during publish, got %v", err)
	}

	close(publisher.block)

	select {
	case err := <-publishDone:
		if err != nil {
			t.Fatalf("publish returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for publish to complete")
	}

	select {
	case err := <-cleanupDone:
		if err != nil {
			t.Fatalf("cleanup returned error after publish: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cleanup to complete")
	}
}

type stubReleaseRecordOutcome struct {
	result catalog.ReleaseRecordResult
	err    error
}

type stubReleaseRecordCreator struct {
	calls   int
	results []stubReleaseRecordOutcome
}

func (s *stubReleaseRecordCreator) CreateReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error) {
	s.calls++
	index := s.calls - 1
	if index >= len(s.results) {
		return catalog.ReleaseRecordResult{}, catalog.NewError(500, "internal_error", "unexpected extra release record call", nil)
	}
	return s.results[index].result, s.results[index].err
}

type capturePackagePublisher struct {
	calls     int
	lastInput artifactadapter.NPMPublishInput
	err       error
	started   chan struct{}
	block     chan struct{}
	once      sync.Once
}

func (s *capturePackagePublisher) PublishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) error {
	s.calls++
	s.lastInput = input
	if s.started != nil {
		s.once.Do(func() {
			close(s.started)
		})
	}
	if s.block != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.block:
		}
	}
	return s.err
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
