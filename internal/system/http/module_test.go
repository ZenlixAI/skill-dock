package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"reflect"
	"testing"
)

type stubPinger struct {
	err error
}

func (p stubPinger) Ping(context.Context) error {
	return p.err
}

func TestModuleReadyzCollectsDependencyChecks(t *testing.T) {
	module := NewModule(
		NamedDependency{Name: "repository", Pinger: stubPinger{err: errors.New("repository unavailable")}},
		NamedDependency{Name: "audit", Pinger: stubPinger{err: errors.New("audit unavailable")}},
		NamedDependency{Name: "idempotency", Pinger: stubPinger{}},
	)

	payload, status := module.Readyz(context.Background())

	if status != stdhttp.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusServiceUnavailable, status)
	}

	wantChecks := map[string]string{
		"repository":  "repository unavailable",
		"audit":       "audit unavailable",
		"idempotency": "ok",
	}
	if payload.Status != "degraded" {
		t.Fatalf("expected degraded status, got %#v", payload)
	}
	if !reflect.DeepEqual(payload.Checks, wantChecks) {
		t.Fatalf("unexpected readiness checks (-want +got):\nwant: %#v\ngot:  %#v", wantChecks, payload.Checks)
	}
}
