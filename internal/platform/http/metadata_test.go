package http

import (
	"context"
	stdhttp "net/http"
	"testing"
)

func TestContextWithRequestMetadataStoresIDs(t *testing.T) {
	ctx := ContextWithRequestMetadata(context.Background(), "req_123", "trace_123")

	if got := RequestIDFromContext(ctx); got != "req_123" {
		t.Fatalf("unexpected request id: %q", got)
	}
	if got := TraceIDFromContext(ctx); got != "trace_123" {
		t.Fatalf("unexpected trace id: %q", got)
	}
}

func TestObservedOutcomeDefaultsFromStatus(t *testing.T) {
	request, err := stdhttp.NewRequest(stdhttp.MethodGet, "/skills", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	request = EnsureObservedState(request)
	state := ObservedStateFromContext(request.Context())
	if state == nil {
		t.Fatal("expected observed state")
	}

	if got := ObservedOutcome(stdhttp.StatusUnauthorized, state); got != "unauthorized" {
		t.Fatalf("unexpected outcome: %q", got)
	}

	SetObservedOutcome(request, "cached")
	if got := ObservedOutcome(stdhttp.StatusOK, state); got != "cached" {
		t.Fatalf("unexpected explicit outcome: %q", got)
	}
}
