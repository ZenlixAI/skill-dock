package catalogapi

import (
	"context"
	"net/url"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/system"
	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

// Consumer-side interfaces the Server depends on. Each cohesive module lives in
// its own sub-package and is wired in through one of these seams, which also
// lets tests substitute lightweight stubs.

type systemWorkflow interface {
	Readyz(ctx context.Context) (system.Readyz, int)
}

type auditQueryWorkflow interface {
	ListEvents(ctx context.Context, principal auth.Principal, values url.Values) (persist.AuditPage, *catalog.Error)
}
