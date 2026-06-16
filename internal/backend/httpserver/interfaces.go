package httpserver

import (
	"context"
	"net/url"

	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
	persist "skill_registry/internal/platform/persistence"
	internalsystemhttp "skill_registry/internal/system/http"
)

// Consumer-side interfaces the Server depends on. Each cohesive module lives in
// its own sub-package and is wired in through one of these seams, which also
// lets tests substitute lightweight stubs.

type systemWorkflow interface {
	Readyz(ctx context.Context) (internalsystemhttp.Readyz, int)
}

type auditQueryWorkflow interface {
	ListEvents(ctx context.Context, principal auth.Principal, values url.Values) (persist.AuditPage, *catalog.Error)
}
