package http

import (
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	catalog "skill_registry/internal/catalog/domain"
)

func TestExecuteRawBodyMutationWritesJSONAfterAuthorizationAndLimiter(t *testing.T) {
	t.Helper()

	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/test", strings.NewReader(`{"ok":true}`))
	recorder := httptest.NewRecorder()

	var (
		authorizeCalled bool
		executeCalled   bool
		released        bool
	)

	ExecuteRawBodyMutation(recorder, request, RawBodyMutationConfig{
		Authorize: func(rawBody []byte) bool {
			authorizeCalled = true
			return string(rawBody) == `{"ok":true}`
		},
		AcquireWriteLimiter: func() (func(), *catalog.Error) {
			return func() {
				released = true
			}, nil
		},
		Execute: func(ctx context.Context, rawBody []byte) (any, int, *catalog.Error) {
			executeCalled = true
			if ctx != request.Context() {
				t.Fatal("expected request context passed to execute")
			}
			return map[string]any{"size": len(rawBody)}, stdhttp.StatusCreated, nil
		},
	})

	if !authorizeCalled {
		t.Fatal("expected authorize hook to be called")
	}
	if !executeCalled {
		t.Fatal("expected execute hook to be called")
	}
	if !released {
		t.Fatal("expected write limiter to be released")
	}
	if recorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusCreated, recorder.Code)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"size":11}` {
		t.Fatalf("unexpected response body %q", got)
	}
}

func TestExecuteRawBodyMutationWritesCatalogErrorWhenBodyReadFails(t *testing.T) {
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/test", io.NopCloser(errReader{err: errors.New("boom")}))
	recorder := httptest.NewRecorder()

	ExecuteRawBodyMutation(recorder, request, RawBodyMutationConfig{})

	if recorder.Code != stdhttp.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusUnprocessableEntity, recorder.Code)
	}
	if got := recorder.Body.String(); !strings.Contains(got, `"code":"invalid_request"`) {
		t.Fatalf("expected invalid_request body, got %q", got)
	}
}

func TestExecuteRawBodyMutationStopsWhenAuthorizationFails(t *testing.T) {
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/test", strings.NewReader(`{}`))
	recorder := httptest.NewRecorder()

	executed := false
	ExecuteRawBodyMutation(recorder, request, RawBodyMutationConfig{
		Authorize: func([]byte) bool {
			return false
		},
		Execute: func(context.Context, []byte) (any, int, *catalog.Error) {
			executed = true
			return map[string]any{"ok": true}, stdhttp.StatusOK, nil
		},
	})

	if executed {
		t.Fatal("expected execute not to be called")
	}
	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected untouched response status %d, got %d", stdhttp.StatusOK, recorder.Code)
	}
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}
