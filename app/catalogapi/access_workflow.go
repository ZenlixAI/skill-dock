package catalogapi

import (
	"log"
	"net/http"

	"skill_registry/app/auth"
	"skill_registry/app/observability"
)

type accessWorkflow struct {
	authorizer *auth.Authorizer
	audit      *AuditRecorder
	logger     *log.Logger
}

type accessWorkflowHandler interface {
	authorizeAction(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool
	authorizePackage(writer http.ResponseWriter, request *http.Request, packageName string) bool
	deny(writer http.ResponseWriter, request *http.Request, current auth.Principal, authErr *auth.AuthorizationError, packageName string, version string)
}

func (w *accessWorkflow) authorizeAction(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool {
	current := auth.PrincipalFromContext(request.Context())
	observability.SetObservedAction(writer, string(action))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	if authErr := w.authorizer.AuthorizeAction(current, action); authErr != nil {
		w.deny(writer, request, current, authErr, packageName, version)
		return false
	}
	return true
}

func (w *accessWorkflow) authorizePackage(writer http.ResponseWriter, request *http.Request, packageName string) bool {
	current := auth.PrincipalFromContext(request.Context())
	if authErr := w.authorizer.AuthorizePackage(current, packageName); authErr != nil {
		w.deny(writer, request, current, authErr, packageName, "")
		return false
	}
	return true
}

func (w *accessWorkflow) deny(writer http.ResponseWriter, request *http.Request, current auth.Principal, authErr *auth.AuthorizationError, packageName string, version string) {
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionDenied)
	observability.SetObservedOutcome(writer, authErr.Code)
	if w.audit != nil {
		if err := w.audit.RecordAuthorizationDenied(request, current, authErr.Code, authErr.Status, packageName, version, authErr.Details); err != nil && w.logger != nil {
			w.logger.Printf(
				`{"event":"audit_write_failed","requestId":%q,"traceId":%q,"action":%q,"authDecision":%q,"error":%q}`,
				requestIDFromContext(request.Context()),
				traceIDFromContext(request.Context()),
				"auth.denied",
				observability.AuthDecisionDenied,
				err.Error(),
			)
		}
	}
	writeAuthorizationError(writer, request, authErr)
}
