package application

import (
	"context"
	"testing"

	catalog "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
	artifactadapter "skill_registry/internal/platform/artifact"
)

func TestPublishServicePublishesWhenArtifactMissing(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	runtime := NewWorkspaceService(root, store)

	releaseRecordCalls := 0
	releaseRecordResults := []struct {
		result catalog.ReleaseRecordResult
		err    error
	}{
		{err: ErrReleaseArtifactMissing},
		{result: catalog.ReleaseRecordResult{Created: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
	}
	createReleaseRecord := func(context.Context, string, string, catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error) {
		index := releaseRecordCalls
		releaseRecordCalls++
		if index >= len(releaseRecordResults) {
			return catalog.ReleaseRecordResult{}, catalog.NewError(500, "internal_error", "unexpected extra release record call", nil)
		}
		return releaseRecordResults[index].result, releaseRecordResults[index].err
	}

	publishCalls := 0
	var lastInput artifactadapter.NPMPublishInput
	publishPackage := func(_ context.Context, input artifactadapter.NPMPublishInput) error {
		publishCalls++
		lastInput = input
		return nil
	}

	service := NewPublishService(runtime, createReleaseRecord, publishPackage)

	response, err := service.PublishContext(context.Background(), packagingdomain.PublishRequest{
		Source: packagingdomain.Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   packagingdomain.Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
		Metadata: packagingdomain.Metadata{DisplayName: "managed", Description: "desc", SourceType: "native"},
		Options:  packagingdomain.Options{},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !response.PublishedToRegistry || !response.Created || response.IdempotentReplay {
		t.Fatalf("unexpected publish response: %+v", response)
	}
	if releaseRecordCalls != 2 {
		t.Fatalf("expected 2 release record attempts, got %d", releaseRecordCalls)
	}
	if publishCalls != 1 {
		t.Fatalf("expected 1 publisher call, got %d", publishCalls)
	}
	if lastInput.TarballPath == "" {
		t.Fatal("expected publish workflow to pass a tarball path")
	}
	if lastInput.Description != "desc" {
		t.Fatalf("expected publish description from workspace file, got %q", lastInput.Description)
	}
	if lastInput.DisplayName != "managed" {
		t.Fatalf("expected publish display name from workspace file, got %q", lastInput.DisplayName)
	}
	if lastInput.SourceType != "native" {
		t.Fatalf("expected publish source type from workspace file, got %q", lastInput.SourceType)
	}
}
