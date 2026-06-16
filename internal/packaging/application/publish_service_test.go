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
	service := NewPublishService(runtime)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{err: ErrReleaseArtifactMissing},
			{result: catalog.ReleaseRecordResult{Created: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{}
	service.Configure(repo, publisher)

	response, err := service.Publish(context.Background(), packagingdomain.PublishRequest{
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
	if repo.calls != 2 {
		t.Fatalf("expected 2 release record attempts, got %d", repo.calls)
	}
	if publisher.calls != 1 {
		t.Fatalf("expected 1 publisher call, got %d", publisher.calls)
	}
	if publisher.lastInput.TarballPath == "" {
		t.Fatal("expected publish workflow to pass a tarball path")
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
}

func (s *capturePackagePublisher) PublishPackage(ctx context.Context, input artifactadapter.NPMPublishInput) error {
	s.calls++
	s.lastInput = input
	return s.err
}
