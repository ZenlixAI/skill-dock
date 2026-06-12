package catalogapi

import (
	"io"
	"net/http"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
)

func (s *Server) handleReleaseRecord(writer http.ResponseWriter, request *http.Request, packageName string, versionPath string) {
	release, limitErr := s.writeLimiter.acquire()
	if limitErr != nil {
		common.WriteCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		common.WriteCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	response, status, err := s.catalogMutationModule.createReleaseRecord(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		packageName,
		versionPath,
		rawBody,
	)
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	common.WriteJSON(writer, status, response)
}

func (s *Server) handleSkillGovernanceChange(writer http.ResponseWriter, request *http.Request, packageName string) {
	release, limitErr := s.writeLimiter.acquire()
	if limitErr != nil {
		common.WriteCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		common.WriteCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	response, status, err := s.catalogMutationModule.applySkillGovernanceChange(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		packageName,
		rawBody,
	)
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	common.WriteJSON(writer, status, response)
}

func (s *Server) handleVersionGovernanceChange(writer http.ResponseWriter, request *http.Request, packageName string, versionPath string) {
	release, limitErr := s.writeLimiter.acquire()
	if limitErr != nil {
		common.WriteCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		common.WriteCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	response, status, err := s.catalogMutationModule.applyVersionGovernanceChange(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		packageName,
		versionPath,
		rawBody,
	)
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	common.WriteJSON(writer, status, response)
}
