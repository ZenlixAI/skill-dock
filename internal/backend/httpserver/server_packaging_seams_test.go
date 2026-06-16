package catalogapi

import (
	"testing"

	packagingapplication "skill_registry/internal/packaging/application"
)

func TestServerAcceptsInternalPackagingApplicationService(t *testing.T) {
	var _ packagingApplication = packagingapplication.NewService("")
}
