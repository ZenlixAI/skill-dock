package catalogapi

import "testing"

func TestRequestLimiterRejectsWhenCapacityIsExceeded(t *testing.T) {
	limiter := newRequestLimiter(1)
	release, err := limiter.acquire()
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	_, err = limiter.acquire()
	if err == nil {
		t.Fatal("expected limiter rejection")
	}
	if err.Code != "request_limit_exceeded" {
		t.Fatalf("unexpected error code: %s", err.Code)
	}
}

func TestRequestLimiterUnlimitedWhenDisabled(t *testing.T) {
	limiter := newRequestLimiter(0)
	release, err := limiter.acquire()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	release()
}
