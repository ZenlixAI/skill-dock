package httpserver

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	internalpackaginghttp "skill_registry/internal/packaging/http"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

func TestPackagingWorkflowUsesPackagingApplicationWorkflowContract(t *testing.T) {
	var _ internalpackaginghttp.WorkflowService = (*packagingapplication.Service)(nil)
}

func TestPackagingWorkflowReplaysCachedPrepareResponse(t *testing.T) {
	workflow := newPackagingWorkflowForTest(t)
	payload := packaging.PrepareRequest{
		Target: packaging.Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	}

	first, firstStatus, firstErr := workflow.Prepare(context.Background(), auth.Principal{Role: "authoring"}, payload)
	second, secondStatus, secondErr := workflow.Prepare(context.Background(), auth.Principal{Role: "authoring"}, payload)

	if firstErr != nil || secondErr != nil {
		t.Fatalf("expected cached success, got firstErr=%v secondErr=%v", firstErr, secondErr)
	}
	if firstStatus != http.StatusOK || secondStatus != http.StatusOK {
		t.Fatalf("expected status 200/200, got %d/%d", firstStatus, secondStatus)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected cached replay body to match")
	}
	if got := workflow.prepareCallCount(); got != 1 {
		t.Fatalf("expected underlying prepare path to execute once, got %d", got)
	}
}

func TestPackagingWorkflowPrepareRecordsSuccessAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := newPackagingWorkflowWithAuditForTest(platformaudit.NewRecorder(store))
	principal := auth.Principal{Role: "authoring", Actor: "svc-authoring", AuthMechanism: "configured_token"}
	ctx := platformhttp.ContextWithRequestMetadata(context.Background(), "req_packaging_success", "trace_packaging_success")

	_, status, err := workflow.Prepare(ctx, principal, packaging.PrepareRequest{
		Target: packaging.Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})

	if err != nil || status != http.StatusOK {
		t.Fatalf("expected prepare success, got status=%d err=%v", status, err)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Action != "packaging.prepare" || store.events[0].Outcome != "succeeded" {
		t.Fatalf("unexpected success audit event: %+v", store.events[0])
	}
}

func TestPackagingWorkflowPrepareRecordsFailureAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := newPackagingWorkflowWithAuditForTest(platformaudit.NewRecorder(store))
	principal := auth.Principal{Role: "authoring", Actor: "svc-authoring", AuthMechanism: "configured_token"}
	ctx := platformhttp.ContextWithRequestMetadata(context.Background(), "req_packaging_failure", "trace_packaging_failure")

	_, status, err := workflow.Prepare(ctx, principal, packaging.PrepareRequest{
		Source: packaging.Source{Type: "local"},
		Target: packaging.Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})

	if err == nil || status != http.StatusUnprocessableEntity {
		t.Fatalf("expected prepare failure, got status=%d err=%v", status, err)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Action != "packaging.prepare" || store.events[0].Outcome != "failed" {
		t.Fatalf("unexpected failure audit event: %+v", store.events[0])
	}
}

func TestPackagingWorkflowDecodePrepareRequestRecordsFailureAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := newPackagingWorkflowWithAuditForTest(platformaudit.NewRecorder(store))
	principal := auth.Principal{Role: "authoring", Actor: "svc-authoring", AuthMechanism: "configured_token"}
	ctx := platformhttp.ContextWithRequestMetadata(context.Background(), "req_packaging_decode_failure", "trace_packaging_decode_failure")

	_, err := workflow.DecodePrepareRequest(ctx, principal, []byte(`{"source":{"type":"local"},"target":{"packageName":"@platform.skill/test","version":"1.0.0"}}{"extra":true}`))

	if err == nil {
		t.Fatal("expected decode error")
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Action != "packaging.prepare" || store.events[0].Outcome != "failed" {
		t.Fatalf("unexpected decode audit event: %+v", store.events[0])
	}
	if store.events[0].StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 audit status, got %+v", store.events[0])
	}
}

func TestPackagingWorkflowPassesRequestContextToPackagingService(t *testing.T) {
	seenRequestID := ""
	workflow := internalpackaginghttp.NewWorkflow(internalpackaginghttp.WorkflowConfig{
		Service: &stubPackagingService{
			prepareFunc: func(ctx context.Context, input packaging.PrepareRequest) (packaging.PrepareResponse, error) {
				value := platformhttp.RequestIDFromContext(ctx)
				seenRequestID = value
				return packaging.PrepareResponse{
					WorkspacePath: "workspace-handle",
					PackageName:   input.Target.PackageName,
					Version:       input.Target.Version,
				}, nil
			},
		},
		CacheTTL:            10 * time.Minute,
		AllowAnonymousCache: true,
	})
	ctx := platformhttp.ContextWithRequestMetadata(context.Background(), "req_packaging_ctx", "")

	_, status, err := workflow.Prepare(ctx, auth.Principal{Role: "authoring"}, packaging.PrepareRequest{
		Target: packaging.Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	})

	if err != nil || status != http.StatusOK {
		t.Fatalf("expected prepare success, got status=%d err=%v", status, err)
	}
	if seenRequestID != "req_packaging_ctx" {
		t.Fatalf("expected request context to reach packaging service, got %q", seenRequestID)
	}
}
