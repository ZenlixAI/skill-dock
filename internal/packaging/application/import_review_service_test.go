package application

import (
	"context"
	"os"
	"testing"

	packagingdomain "skill_registry/internal/packaging/domain"
)

func TestImportReviewServiceReviewReturnsReviewDataAndCleansWorkspace(t *testing.T) {
	root := t.TempDir()
	service := NewImportReviewService(NewWorkspaceService(root, nil))

	response, err := service.Review(context.Background(), packagingdomain.ImportReviewRequest{
		Source: packagingdomain.Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target: packagingdomain.Target{
			PackageName: "@platform.skill/review-fixture",
			Version:     "0.4.0",
		},
		Metadata: packagingdomain.Metadata{
			DisplayName: "Review Fixture",
			Description: "Uploaded archive",
			SourceType:  "native",
		},
		Options: packagingdomain.Options{},
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}

	if !response.OK {
		t.Fatalf("expected ok response, got %+v", response)
	}
	if response.PackageName != "@platform.skill/review-fixture" || response.Version != "0.4.0" {
		t.Fatalf("unexpected identity: %+v", response)
	}
	if response.ManifestDigest == "" {
		t.Fatalf("expected manifest digest, got %+v", response)
	}
	if len(response.Components) == 0 {
		t.Fatalf("expected components, got %+v", response)
	}
	if response.BundleBase64 != "" {
		t.Fatalf("expected no bundle payload by default, got %+v", response)
	}
	if len(response.Errors) != 0 {
		t.Fatalf("expected no validation errors, got %+v", response)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read temp root: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected temporary workspace cleanup, found %d entries", len(entries))
	}
}

func TestImportReviewServiceReviewIncludesBundleWhenRequested(t *testing.T) {
	root := t.TempDir()
	service := NewImportReviewService(NewWorkspaceService(root, nil))

	response, err := service.Review(context.Background(), packagingdomain.ImportReviewRequest{
		Source: packagingdomain.Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target: packagingdomain.Target{
			PackageName: "@platform.skill/review-fixture",
			Version:     "0.4.0",
		},
		Metadata: packagingdomain.Metadata{
			DisplayName: "Review Fixture",
			Description: "Uploaded archive",
			SourceType:  "native",
		},
		Options:             packagingdomain.Options{},
		IncludeBundleBase64: true,
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if response.BundleBase64 == "" {
		t.Fatalf("expected bundle payload, got %+v", response)
	}
}
