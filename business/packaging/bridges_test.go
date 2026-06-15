package packaging

import (
	"reflect"
	"testing"

	packagingdomain "skill_registry/internal/packaging/domain"
)

func TestLegacyPackagingComponentsBridgeToInternalImplementations(t *testing.T) {
	if newBundleRuleModule().impl == nil {
		t.Fatal("expected bundle rule module bridge to hold an internal implementation")
	}
	if newWorkspaceSourceModule(t.TempDir(), 0).impl == nil {
		t.Fatal("expected workspace source module bridge to hold an internal implementation")
	}
	runtime := newPackagingWorkspaceRuntime(t.TempDir(), nil)
	if runtime.impl == nil {
		t.Fatal("expected workspace runtime bridge to hold an internal implementation")
	}
	if newPublishedReleaseWorkflow(runtime).impl == nil {
		t.Fatal("expected publish workflow bridge to hold an internal implementation")
	}
	if !reflect.DeepEqual(
		warningDetails([]string{"normalized_missing_package_json", "missing_capability_content"}),
		packagingdomain.WarningDetails([]string{"normalized_missing_package_json", "missing_capability_content"}),
	) {
		t.Fatal("expected warning detail bridge to preserve behavior")
	}
}
