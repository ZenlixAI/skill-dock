package access

import (
	"log"
	"net/http"

	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	"skill_registry/internal/platform/observability"
)

type Gate interface {
	AuthorizeAction(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool
	AuthorizePackage(writer http.ResponseWriter, request *http.Request, packageName string) bool
	Deny(writer http.ResponseWriter, request *http.Request, current auth.Principal, authErr *auth.AuthorizationError, packageName string, version string)
}

type Workflow struct {
	authorizer *auth.Authorizer
	audit      *platformaudit.Recorder
	logger     *log.Logger
}

func New(authorizer *auth.Authorizer, audit *platformaudit.Recorder, logger *log.Logger) *Workflow {
	return &Workflow{
		authorizer: authorizer,
		audit:      audit,
		logger:     logger,
	}
}

func (w *Workflow) AuthorizeAction(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool {
	current := auth.PrincipalFromContext(request.Context())
	platformhttp.SetObservedAction(request, string(action))
	platformhttp.SetObservedAuthDecision(request, observability.AuthDecisionAllowed)
	if authErr := w.authorizer.AuthorizeAction(current, action); authErr != nil {
		w.Deny(writer, request, current, authErr, packageName, version)
		return false
	}
	return true
}

func (w *Workflow) AuthorizePackage(writer http.ResponseWriter, request *http.Request, packageName string) bool {
	current := auth.PrincipalFromContext(request.Context())
	if authErr := w.authorizer.AuthorizePackage(current, packageName); authErr != nil {
		w.Deny(writer, request, current, authErr, packageName, "")
		return false
	}
	return true
}

func (w *Workflow) Deny(writer http.ResponseWriter, request *http.Request, current auth.Principal, authErr *auth.AuthorizationError, packageName string, version string) {
	platformhttp.SetObservedAuthDecision(request, observability.AuthDecisionDenied)
	platformhttp.SetObservedOutcome(request, authErr.Code)
	if w.audit != nil {
		if err := w.audit.RecordAuthorizationDenied(request, current, authErr.Code, authErr.Status, packageName, version, authErr.Details); err != nil && w.logger != nil {
			w.logger.Printf(
				`{"event":"audit_write_failed","requestId":%q,"traceId":%q,"action":%q,"authDecision":%q,"error":%q}`,
				platformhttp.RequestIDFromContext(request.Context()),
				platformhttp.TraceIDFromContext(request.Context()),
				"auth.denied",
				observability.AuthDecisionDenied,
				err.Error(),
			)
		}
	}
	platformhttp.WriteAuthorizationError(writer, request, authErr)
}
