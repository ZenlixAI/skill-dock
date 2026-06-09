package packaging

import (
	"context"
	"fmt"
	"testing"
	"time"

	"skill_registry/business/catalog"
)

func TestPublishedReleaseWorkflowPublishesWhenArtifactMissing(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	runtime := newPackagingWorkspaceRuntime(root, store)
	workflow := newPublishedReleaseWorkflow(runtime)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{err: errReleaseArtifactMissing},
			{result: catalog.ReleaseRecordResult{Created: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{}
	workflow.Configure(repo, publisher)

	prepareResp, err := runtime.Prepare(context.Background(), PrepareRequest{
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
	if err := store.SavePackageMetadata(context.Background(), prepareResp.WorkspacePath, []byte(`{"name":"@platform.skill/managed","version":"1.0.0","description":123,"skill":{"displayName":456,"sourceType":"native"}}`), time.Now().UTC()); err != nil {
		t.Fatalf("save invalid cached metadata: %v", err)
	}

	response, err := workflow.Publish(context.Background(), PublishRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
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
	if publisher.lastInput.Description != "desc" {
		t.Fatalf("expected publish description from workspace file, got %q", publisher.lastInput.Description)
	}
	if publisher.lastInput.DisplayName != "managed" {
		t.Fatalf("expected publish display name from workspace file, got %q", publisher.lastInput.DisplayName)
	}
	if publisher.lastInput.SourceType != "native" {
		t.Fatalf("expected publish source type from workspace file, got %q", publisher.lastInput.SourceType)
	}
}

func TestPublishedReleaseWorkflowSkipsRegistryPublishWhenReleaseAlreadyExists(t *testing.T) {
	runtime := newPackagingWorkspaceRuntime(t.TempDir(), nil)
	workflow := newPublishedReleaseWorkflow(runtime)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{result: catalog.ReleaseRecordResult{Created: false, IdempotentReplay: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{}
	workflow.Configure(repo, publisher)

	sourceDir := copyExternalSkillFixture(t, "minimal-imported-skill")
	prepareResp, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	response, err := workflow.Publish(context.Background(), PublishRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if response.PublishedToRegistry || response.Created || !response.IdempotentReplay {
		t.Fatalf("unexpected publish response: %+v", response)
	}
	if repo.calls != 1 {
		t.Fatalf("expected 1 release record attempt, got %d", repo.calls)
	}
	if publisher.calls != 0 {
		t.Fatalf("expected publisher to be skipped, got %d calls", publisher.calls)
	}
}

func TestPublishedReleaseWorkflowReplaysWhenArtifactAlreadyExists(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	runtime := newPackagingWorkspaceRuntime(root, store)
	workflow := newPublishedReleaseWorkflow(runtime)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{err: errReleaseArtifactMissing},
			{result: catalog.ReleaseRecordResult{Created: false, IdempotentReplay: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{
		err: errPackageArtifactAlreadyExists,
	}
	workflow.Configure(repo, publisher)

	prepareResp, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/conflict", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "conflict", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	response, err := workflow.Publish(context.Background(), PublishRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/conflict", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if response.PublishedToRegistry || response.Created || !response.IdempotentReplay {
		t.Fatalf("unexpected publish response: %+v", response)
	}
	if repo.calls != 2 {
		t.Fatalf("expected release record retry after conflict, got %d calls", repo.calls)
	}
	if publisher.calls != 1 {
		t.Fatalf("expected one publisher attempt before replay, got %d", publisher.calls)
	}
}

func TestClassifyReleaseRecordErrorTreatsWrappedCatalogArtifactMissingAsMissing(t *testing.T) {
	state, catalogErr := classifyReleaseRecordError(wrappedReleaseArtifactMissingError())
	if state != releaseRecordWorkflowStateArtifactMissing {
		t.Fatalf("expected artifact-missing state, got %v", state)
	}
	if catalogErr == nil || catalogErr.Code != "registry_version_not_found" {
		t.Fatalf("expected registry_version_not_found error, got %#v", catalogErr)
	}
}

func TestClassifyPackagePublishErrorTreatsWrappedCatalogConflictAsAlreadyExists(t *testing.T) {
	state, catalogErr := classifyPackagePublishError(wrappedPackageArtifactAlreadyExistsError())
	if state != packagePublishWorkflowStateArtifactAlreadyExists {
		t.Fatalf("expected artifact-already-exists state, got %v", state)
	}
	if catalogErr == nil || catalogErr.Code != "registry_version_conflict" {
		t.Fatalf("expected registry_version_conflict error, got %#v", catalogErr)
	}
}

func wrappedReleaseArtifactMissingError() error {
	return fmt.Errorf("release artifact missing state: %w", catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil))
}

func wrappedPackageArtifactAlreadyExistsError() error {
	return fmt.Errorf("package artifact already exists state: %w", catalog.NewError(409, "registry_version_conflict", "Registry version already exists.", nil))
}
