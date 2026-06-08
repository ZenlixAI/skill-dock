package catalogapi

import (
	"net/http"

	"skill_registry/business/packaging"
)

func (s *Server) handlePackagingPublish(writer http.ResponseWriter, request *http.Request) {
	if s.packaging == nil {
		http.NotFound(writer, request)
		return
	}

	var payload packaging.PublishRequest
	if err := decodeJSONBody(request, &payload); err != nil {
		s.recordMutationAudit(request, "packaging.publish", "", "", err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		writeCatalogError(writer, request, err)
		return
	}

	if !s.authorizePackageAccess(writer, request, payload.Target.PackageName) {
		return
	}

	response, err := s.packaging.PublishContext(request.Context(), payload)
	if err != nil {
		catalogErr := catalogErrorFromError(err)
		s.recordMutationAudit(request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, catalogErr.Status, "failed", map[string]any{
			"errorCode": catalogErr.Code,
		})
		writeCatalogError(writer, request, catalogErr)
		return
	}
	status := http.StatusOK
	if response.Created {
		status = http.StatusCreated
	}

	s.recordMutationAudit(request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, status, "succeeded", map[string]any{
		"publishedToRegistry": response.PublishedToRegistry,
		"created":             response.Created,
		"idempotentReplay":    response.IdempotentReplay,
	})
	writeJSON(writer, status, response)
}
