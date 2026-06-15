package observability

import (
	"reflect"
	"testing"

	platformobservability "skill_registry/internal/platform/observability"
)

func TestRecorderAliasMatchesPlatformType(t *testing.T) {
	if reflect.TypeOf(AuthDecision("")) != reflect.TypeOf(platformobservability.AuthDecision("")) {
		t.Fatalf("expected AuthDecision to alias platform AuthDecision")
	}
	if reflect.TypeOf((*StatusRecorder)(nil)) != reflect.TypeOf((*platformobservability.StatusRecorder)(nil)) {
		t.Fatalf("expected StatusRecorder to alias platform StatusRecorder")
	}
	if reflect.TypeOf((*Collector)(nil)) != reflect.TypeOf((*platformobservability.Collector)(nil)) {
		t.Fatalf("expected Collector to alias platform Collector")
	}
}
