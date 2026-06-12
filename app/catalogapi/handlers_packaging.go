package catalogapi

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
)

func (s *Server) handlePackagingPrepare(writer http.ResponseWriter, request *http.Request) {
	handlePackagingEndpoint(s, writer, request, s.packagingModule.decodePrepareRequest, s.packagingModule.prepare)
}

func (s *Server) handlePackagingValidate(writer http.ResponseWriter, request *http.Request) {
	handlePackagingEndpoint(s, writer, request, s.packagingModule.decodeValidateRequest, s.packagingModule.validate)
}

func (s *Server) handlePackagingPack(writer http.ResponseWriter, request *http.Request) {
	handlePackagingEndpoint(s, writer, request, s.packagingModule.decodePackRequest, s.packagingModule.pack)
}

func (s *Server) handlePackagingCleanup(writer http.ResponseWriter, request *http.Request) {
	if s.packaging == nil {
		http.NotFound(writer, request)
		return
	}

	var payload packaging.CleanupRequest
	if err := common.DecodeJSONBody(request, &payload); err != nil {
		s.recordMutationAudit(request, "packaging.cleanup", "", "", err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		common.WriteCatalogError(writer, request, err)
		return
	}

	response, status, err := s.packagingAccessModule.cleanup(request.Context(), auth.PrincipalFromContext(request.Context()), payload)
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	common.WriteJSON(writer, status, response)
}

func (s *Server) handlePackagingBundleDownload(writer http.ResponseWriter, request *http.Request) {
	if s.packaging == nil {
		http.NotFound(writer, request)
		return
	}

	packageName := strings.TrimSpace(request.URL.Query().Get("packageName"))
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}

	bundlePath, err := s.packagingAccessModule.downloadBundle(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		strings.TrimSpace(request.URL.Query().Get("workspacePath")),
		packageName,
		strings.TrimSpace(request.URL.Query().Get("version")),
	)
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}

	bundleFile, openErr := os.Open(bundlePath)
	if openErr != nil {
		common.WriteCatalogError(writer, request, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil))
		return
	}
	defer bundleFile.Close()

	info, statErr := bundleFile.Stat()
	if statErr != nil || info.IsDir() {
		common.WriteCatalogError(writer, request, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil))
		return
	}

	writer.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(writer, request, filepath.Base(bundlePath), info.ModTime(), bundleFile)
}

func handlePackagingEndpoint[T interface {
	GetIdempotencyKey() string
	GetTarget() packaging.Target
}](s *Server, writer http.ResponseWriter, request *http.Request, decode func(context.Context, auth.Principal, []byte) (T, *catalog.Error), execute func(context.Context, auth.Principal, T) (any, int, *catalog.Error)) {
	if s.packagingModule == nil || s.packagingModule.service == nil {
		http.NotFound(writer, request)
		return
	}

	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		common.WriteCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	principal := auth.PrincipalFromContext(request.Context())
	payload, err := decode(request.Context(), principal, rawBody)
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	if packageName := payload.GetTarget().PackageName; packageName != "" {
		if !s.authorizePackageAccess(writer, request, packageName) {
			return
		}
	}
	release, limitErr := s.writeLimiter.acquire()
	if limitErr != nil {
		common.WriteCatalogError(writer, request, limitErr)
		return
	}
	defer release()
	response, status, err := execute(request.Context(), principal, payload)
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}
	common.WriteJSON(writer, status, response)
}
