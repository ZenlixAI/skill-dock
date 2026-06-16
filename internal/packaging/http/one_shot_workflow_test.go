package http

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

func TestOneShotWorkflowUsesPackagingApplicationReviewContract(t *testing.T) {
	var _ ReviewService = (*packagingapplication.Service)(nil)
}

func TestOneShotWorkflowReplaysCachedImportReviewResponse(t *testing.T) {
	workflow := newOneShotWorkflowForTest(t)
	payload := packaging.ImportReviewRequest{
		Target: packaging.Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	}

	first, firstStatus, firstErr := workflow.ImportReview(context.Background(), auth.Principal{Role: "authoring"}, payload)
	second, secondStatus, secondErr := workflow.ImportReview(context.Background(), auth.Principal{Role: "authoring"}, payload)

	if firstErr != nil || secondErr != nil {
		t.Fatalf("expected cached success, got firstErr=%v secondErr=%v", firstErr, secondErr)
	}
	if firstStatus != http.StatusOK || secondStatus != http.StatusOK {
		t.Fatalf("expected status 200/200, got %d/%d", firstStatus, secondStatus)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected cached replay body to match")
	}
	if got := workflow.reviewCallCount(); got != 1 {
		t.Fatalf("expected underlying review path to execute once, got %d", got)
	}
}

func TestOneShotWorkflowImportReviewRecordsSuccessAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := newOneShotWorkflowWithAuditForTest(platformaudit.NewRecorder(store))
	principal := auth.Principal{Role: "authoring", Actor: "svc-authoring", AuthMechanism: "configured_token"}
	ctx := platformhttp.ContextWithRequestMetadata(context.Background(), "req_import_review_success", "trace_import_review_success")

	_, status, err := workflow.ImportReview(ctx, principal, packaging.ImportReviewRequest{
		Target: packaging.Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})

	if err != nil || status != http.StatusOK {
		t.Fatalf("expected review success, got status=%d err=%v", status, err)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Action != "packaging.import-review" || store.events[0].Outcome != "succeeded" {
		t.Fatalf("unexpected success audit event: %+v", store.events[0])
	}
}

type testOneShotWorkflow struct {
	*OneShotWorkflow
	service *stubReviewService
}

type stubReviewService struct {
	reviewCalls int
	reviewFunc  func(context.Context, packaging.ImportReviewRequest) (packaging.ImportReviewResponse, error)
}

func (s *stubReviewService) ReviewContext(ctx context.Context, input packaging.ImportReviewRequest) (packaging.ImportReviewResponse, error) {
	s.reviewCalls++
	if s.reviewFunc != nil {
		return s.reviewFunc(ctx, input)
	}
	return packaging.ImportReviewResponse{}, nil
}

func newOneShotWorkflowForTest(_ interface{ Helper() }) *testOneShotWorkflow {
	return newOneShotWorkflowWithAuditForTest(nil)
}

func newOneShotWorkflowWithAuditForTest(auditRecorder *platformaudit.Recorder) *testOneShotWorkflow {
	service := &stubReviewService{
		reviewFunc: func(_ context.Context, input packaging.ImportReviewRequest) (packaging.ImportReviewResponse, error) {
			return packaging.ImportReviewResponse{
				OK:             true,
				PackageName:    input.Target.PackageName,
				Version:        input.Target.Version,
				ManifestDigest: "sha256:test",
				Components:     []string{"commands"},
				Warnings:       []string{},
				Errors:         []string{},
			}, nil
		},
	}
	return &testOneShotWorkflow{
		OneShotWorkflow: NewOneShotWorkflow(OneShotWorkflowConfig{
			ReviewService:       service,
			Audit:               auditRecorder,
			CacheTTL:            10 * time.Minute,
			AllowAnonymousCache: true,
		}),
		service: service,
	}
}

func (m *testOneShotWorkflow) reviewCallCount() int {
	return m.service.reviewCalls
}
