package catalogapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"skill_registry/app/auth"
	"skill_registry/app/observability"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	artifactadapter "skill_registry/foundation/artifact"
	"skill_registry/foundation/persist"
)

type ServerConfig struct {
	Repository           catalog.Repository
	ReaderToken          string
	AuthoringToken       string
	PublisherToken       string
	PublicNPMRegistryURL string
	AuthTokens           map[string]auth.TokenConfig
	SessionSecret        string
	SessionUsers         map[string]auth.SessionUserConfig
	MockRegistry         *artifactadapter.MockRegistry
	DisableMockRegistry  bool
	Packaging            *packaging.Service
	Idempotency          persist.IdempotencyStore
	Audit                persist.AuditStore
	Logger               *log.Logger
}

type Server struct {
	repo                 catalog.Repository
	authorizer           *auth.Authorizer
	sessionManager       *auth.SessionManager
	catalogReadModule    catalogReadWorkflow
	governanceModule     *governanceModule
	packagingModule      *packagingModule
	sessionModule        sessionWorkflow
	publicNPMRegistryURL string
	mockRegistry         *artifactadapter.MockRegistry
	disableMockRegistry  bool
	packaging            *packaging.Service
	idempotency          persist.IdempotencyStore
	audit                persist.AuditStore
	auditRecorder        *AuditRecorder
	logger               *log.Logger
	metrics              *observability.Collector
	requestID            atomic.Uint64
}

type contextKey string

const (
	requestIDContextKey contextKey = "request_id"
	traceIDContextKey   contextKey = "trace_id"
)

func NewServer(config ServerConfig) *Server {
	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "", 0)
	}

	authorizer := auth.NewAuthorizer(auth.Config{
		ReaderToken:    config.ReaderToken,
		AuthoringToken: config.AuthoringToken,
		PublisherToken: config.PublisherToken,
		Tokens:         config.AuthTokens,
	})
	sessionManager := auth.NewSessionManager(config.SessionSecret, config.SessionUsers, 12*time.Hour)
	auditRecorder := NewAuditRecorder(config.Audit)
	var packagingSvc packagingService
	if config.Packaging != nil {
		packagingSvc = config.Packaging
	}

	server := &Server{
		repo:                 config.Repository,
		authorizer:           authorizer,
		sessionManager:       sessionManager,
		publicNPMRegistryURL: strings.TrimRight(strings.TrimSpace(config.PublicNPMRegistryURL), "/"),
		mockRegistry:         config.MockRegistry,
		disableMockRegistry:  config.DisableMockRegistry,
		packaging:            config.Packaging,
		idempotency:          config.Idempotency,
		audit:                config.Audit,
		auditRecorder:        auditRecorder,
		logger:               logger,
		metrics:              observability.NewCollector(),
	}
	server.catalogReadModule = &catalogReadModule{
		repo: config.Repository,
	}
	server.governanceModule = &governanceModule{
		repo:       config.Repository,
		authorizer: authorizer,
	}
	server.packagingModule = &packagingModule{
		service:             packagingSvc,
		idempotency:         config.Idempotency,
		audit:               auditRecorder,
		cache:               newPackagingResponseCache(10 * time.Minute),
		allowAnonymousCache: false,
	}
	server.sessionModule = &sessionModule{
		authorizer:     authorizer,
		sessionManager: sessionManager,
	}
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(s.handleHealthz))
	mux.Handle("/readyz", http.HandlerFunc(s.handleReadyz))
	mux.Handle("/metrics", http.HandlerFunc(s.handleMetrics))
	mux.Handle("/api/v1/audit-events", s.requireAction(s.handleAuditEvents, auth.ActionReadAuditEvents))
	mux.Handle("/api/v1/session", http.HandlerFunc(s.handleSession))
	mux.Handle("/api/v1/session/login", http.HandlerFunc(s.handleSessionLogin))
	mux.Handle("/api/v1/session/logout", http.HandlerFunc(s.handleSessionLogout))
	mux.Handle("/api/v1/skills", s.requireAction(s.handleSkills, auth.ActionReadCatalog))
	mux.Handle("/api/v1/skills/search", s.requireAction(s.handleSkills, auth.ActionReadCatalog))
	mux.Handle("/api/v1/skills/curated", s.requireAction(s.handleCuratedSkills, auth.ActionReadCatalog))
	mux.Handle("/api/v1/governance-changes", s.requireAction(s.handleGovernanceChanges, auth.ActionReadCatalog))
	mux.Handle("/api/v1/skills/", s.requireAction(s.handleSkillSubtree, auth.ActionReadCatalog))
	mux.Handle("/api/v1/skill-packaging/prepare", s.requireAction(s.handlePackagingPrepare, auth.ActionPackagingPrepare))
	mux.Handle("/api/v1/skill-packaging/validate", s.requireAction(s.handlePackagingValidate, auth.ActionPackagingValidate))
	mux.Handle("/api/v1/skill-packaging/pack", s.requireAction(s.handlePackagingPack, auth.ActionPackagingPack))
	mux.Handle("/api/v1/skill-packaging/cleanup", s.requireAction(s.handlePackagingCleanup, auth.ActionPackagingPack))
	mux.Handle("/api/v1/skill-packaging/bundle", s.requireAction(s.handlePackagingBundleDownload, auth.ActionPackagingPack))
	if !s.disableMockRegistry {
		mux.Handle("/mock-npm/", http.HandlerFunc(s.handleMockRegistry))
	}
	return s.withCORS(s.withRequestMetadata(s.withObservability(mux)))
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := strings.TrimSpace(request.Header.Get("Origin"))
		if origin != "" && isLocalOperatorOrigin(origin) {
			writer.Header().Set("Access-Control-Allow-Origin", origin)
			writer.Header().Set("Vary", "Origin")
			writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Actor, X-Role, X-Trace-Id")
			writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			writer.Header().Set("Access-Control-Allow-Credentials", "true")
			writer.Header().Set("Access-Control-Max-Age", "600")
		}

		if request.Method == http.MethodOptions && origin != "" && isLocalOperatorOrigin(origin) {
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

func isLocalOperatorOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "127.0.0.1" || host == "localhost"
}

func (s *Server) withRequestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := "req_" + strconv.FormatUint(s.requestID.Add(1), 10)
		writer.Header().Set("X-Request-Id", requestID)
		traceID := strings.TrimSpace(request.Header.Get("X-Trace-Id"))
		if traceID == "" {
			traceID = "trace_" + strings.TrimPrefix(requestID, "req_")
		}
		writer.Header().Set("X-Trace-Id", traceID)
		ctx := context.WithValue(request.Context(), requestIDContextKey, requestID)
		ctx = context.WithValue(ctx, traceIDContextKey, traceID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func (s *Server) withObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		recorder := observability.NewStatusRecorder(writer)
		next.ServeHTTP(recorder, request)
		duration := time.Since(startedAt)
		s.metrics.RecordRequest(
			request.Method,
			request.URL.EscapedPath(),
			recorder.Status,
			duration,
			recorder.ObservedAction,
			recorder.ObservedAuthDecision,
			recorder.Outcome(),
		)
		s.logRequest(request, recorder, startedAt)
	})
}

func (s *Server) requireAction(next http.HandlerFunc, action auth.Action) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		observability.SetObservedAction(writer, string(action))
		authPrincipal, ok := s.authenticateRequest(request)
		if !ok {
			observability.SetObservedAuthDecision(writer, observability.AuthDecisionDenied)
			observability.SetObservedOutcome(writer, "unauthorized")
			writeCatalogError(writer, request, catalog.NewError(401, "unauthorized", "Authentication is required.", nil))
			s.recordAuthenticationDeniedAudit(request)
			return
		}
		request = request.WithContext(auth.ContextWithPrincipal(request.Context(), authPrincipal))
		observability.SetObservedPrincipal(writer, authPrincipal)
		observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
		if authErr := s.authorizer.AuthorizeAction(authPrincipal, action); authErr != nil {
			s.denyAuthorization(writer, request, authPrincipal, authErr, "", "")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) authenticateRequest(request *http.Request) (auth.Principal, bool) {
	if current, ok := s.authorizer.Authenticate(request); ok {
		return current, true
	}
	if current, ok := s.sessionManager.Authenticate(request); ok {
		return current, true
	}
	return auth.Principal{}, false
}

func (s *Server) handleSkills(writer http.ResponseWriter, request *http.Request) {
	if packageName := request.URL.Query().Get("package_name"); packageName != "" {
		if !s.authorizePackageAccess(writer, request, packageName) {
			return
		}
	}
	sourceType, sourceTypeErr := parseSourceTypeFilter(request.URL.Query().Get("source_type"))
	if sourceTypeErr != nil {
		writeCatalogError(writer, request, sourceTypeErr)
		return
	}
	includeRetired, includeRetiredErr := parseBooleanQueryParam(request.URL.Query().Get("include_retired"), "include_retired", false)
	if includeRetiredErr != nil {
		writeCatalogError(writer, request, includeRetiredErr)
		return
	}
	sortValue, sortErr := parseSkillsSort(request.URL.Query().Get("sort"))
	if sortErr != nil {
		writeCatalogError(writer, request, sortErr)
		return
	}
	pageSize, err := parsePageSize(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	cursor, err := parseCursor(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}

	response, repoErr := s.catalogReadModule.listSkills(request.Context(), auth.PrincipalFromContext(request.Context()), catalog.SkillQuery{
		Query:          request.URL.Query().Get("q"),
		PackageName:    request.URL.Query().Get("package_name"),
		SourceType:     sourceType,
		IncludeRetired: includeRetired,
		Sort:           sortValue,
		PageSize:       pageSize,
		Cursor:         cursor,
	})
	if repoErr != nil {
		writeCatalogError(writer, request, repoErr)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleCuratedSkills(writer http.ResponseWriter, request *http.Request) {
	pageSize, err := parsePageSize(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	cursor, err := parseCursor(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}

	response, repoErr := s.catalogReadModule.listCuratedSkills(request.Context(), auth.PrincipalFromContext(request.Context()), pageSize, cursor)
	if repoErr != nil {
		writeCatalogError(writer, request, repoErr)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleHealthz(writer http.ResponseWriter, request *http.Request) {
	observability.SetObservedAction(writer, string(auth.ActionReadHealthz))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	writeJSON(writer, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) handleReadyz(writer http.ResponseWriter, request *http.Request) {
	observability.SetObservedAction(writer, string(auth.ActionReadReadyz))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	checks := map[string]string{
		"repository":  "ok",
		"audit":       "ok",
		"idempotency": "ok",
	}
	status := http.StatusOK

	if pinger, ok := s.repo.(readinessPinger); ok {
		if err := pinger.Ping(request.Context()); err != nil {
			checks["repository"] = err.Error()
			status = http.StatusServiceUnavailable
		}
	}
	if pinger, ok := s.audit.(readinessPinger); ok {
		if err := pinger.Ping(request.Context()); err != nil {
			checks["audit"] = err.Error()
			status = http.StatusServiceUnavailable
		}
	}
	if pinger, ok := s.idempotency.(readinessPinger); ok {
		if err := pinger.Ping(request.Context()); err != nil {
			checks["idempotency"] = err.Error()
			status = http.StatusServiceUnavailable
		}
	}

	payload := map[string]any{
		"status": "ready",
		"checks": checks,
	}
	if status != http.StatusOK {
		payload["status"] = "degraded"
	}
	writeJSON(writer, status, payload)
}

func (s *Server) handleMetrics(writer http.ResponseWriter, request *http.Request) {
	observability.SetObservedAction(writer, string(auth.ActionReadMetrics))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(s.metrics.RenderPrometheus()))
}

func (s *Server) handleAuditEvents(writer http.ResponseWriter, request *http.Request) {
	if s.audit == nil {
		writeCatalogError(writer, request, catalog.NewError(503, "internal_error", "Audit store is not configured.", nil))
		return
	}

	pageSize, err := parsePageSize(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	cursor, err := parseCursor(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}

	packageName := request.URL.Query().Get("package_name")
	if packageName != "" && !s.authorizePackageAccess(writer, request, packageName) {
		return
	}

	current := auth.PrincipalFromContext(request.Context())
	if authErr := s.authorizer.RequirePackageFilter(current, packageName); authErr != nil {
		writeAuthorizationError(writer, request, authErr)
		return
	}

	authDecision, authDecisionErr := parseAuthDecisionFilter(request.URL.Query().Get("auth_decision"))
	if authDecisionErr != nil {
		writeCatalogError(writer, request, authDecisionErr)
		return
	}
	versionFilter, versionFilterErr := parseExactVersionQueryParam(request.URL.Query().Get("version"), "version")
	if versionFilterErr != nil {
		writeCatalogError(writer, request, versionFilterErr)
		return
	}

	response, auditErr := s.audit.List(request.Context(), persist.AuditQuery{
		Action:       request.URL.Query().Get("action"),
		Actor:        request.URL.Query().Get("actor"),
		AuthDecision: authDecision,
		PackageName:  packageName,
		Version:      versionFilter,
		Outcome:      request.URL.Query().Get("outcome"),
		PageSize:     pageSize,
		Cursor:       cursor,
	})
	if auditErr != nil {
		writeCatalogError(writer, request, auditErr)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleGovernanceChanges(writer http.ResponseWriter, request *http.Request) {
	pageSize, err := parsePageSize(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	cursor, err := parseCursor(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	if packageName := request.URL.Query().Get("package_name"); packageName != "" {
		if !s.authorizePackageAccess(writer, request, packageName) {
			return
		}
	}
	targetType, targetTypeErr := parseGovernanceTargetTypeFilter(request.URL.Query().Get("target_type"))
	if targetTypeErr != nil {
		writeCatalogError(writer, request, targetTypeErr)
		return
	}
	versionFilter, versionFilterErr := parseExactVersionQueryParam(request.URL.Query().Get("version"), "version")
	if versionFilterErr != nil {
		writeCatalogError(writer, request, versionFilterErr)
		return
	}

	response, repoErr := s.governanceModule.listChanges(request.Context(), auth.PrincipalFromContext(request.Context()), catalog.GovernanceQuery{
		PackageName: request.URL.Query().Get("package_name"),
		Version:     versionFilter,
		TargetType:  targetType,
		PageSize:    pageSize,
		Cursor:      cursor,
	})
	if repoErr != nil {
		writeCatalogError(writer, request, repoErr)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleSkillSubtree(writer http.ResponseWriter, request *http.Request) {
	packageName, tail, err := splitSkillPath(request.URL)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}

	switch {
	case len(tail) == 0 && request.Method == http.MethodGet:
		s.handleSkillDetail(writer, request, packageName)
	case len(tail) == 1 && tail[0] == "versions" && request.Method == http.MethodGet:
		s.handleVersionList(writer, request, packageName)
	case len(tail) == 1 && tail[0] == "install-target" && request.Method == http.MethodGet:
		s.handleInstallTarget(writer, request, packageName)
	case len(tail) == 1 && tail[0] == "governance-changes" && request.Method == http.MethodPost:
		if !s.authorizeActionAccess(writer, request, auth.ActionApplySkillGovernance, packageName, "") {
			return
		}
		s.handleSkillGovernanceChange(writer, request, packageName)
	case len(tail) == 2 && tail[0] == "versions" && request.Method == http.MethodGet:
		s.handleVersionDetail(writer, request, packageName, tail[1])
	case len(tail) == 3 && tail[0] == "versions" && tail[2] == "release-record" && request.Method == http.MethodPut:
		if !s.authorizeActionAccess(writer, request, auth.ActionCreateReleaseRecord, packageName, tail[1]) {
			return
		}
		s.handleReleaseRecord(writer, request, packageName, tail[1])
	case len(tail) == 3 && tail[0] == "versions" && tail[2] == "governance-changes" && request.Method == http.MethodPost:
		if !s.authorizeActionAccess(writer, request, auth.ActionApplyVersionGovernance, packageName, tail[1]) {
			return
		}
		s.handleVersionGovernanceChange(writer, request, packageName, tail[1])
	default:
		http.NotFound(writer, request)
	}
}

func (s *Server) handleSkillDetail(writer http.ResponseWriter, request *http.Request, packageName string) {
	response, err := s.repo.GetSkill(request.Context(), packageName)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleVersionList(writer http.ResponseWriter, request *http.Request, packageName string) {
	includeDeprecated, includeDeprecatedErr := parseBooleanQueryParam(request.URL.Query().Get("include_deprecated"), "include_deprecated", true)
	if includeDeprecatedErr != nil {
		writeCatalogError(writer, request, includeDeprecatedErr)
		return
	}
	includeYanked, includeYankedErr := parseBooleanQueryParam(request.URL.Query().Get("include_yanked"), "include_yanked", false)
	if includeYankedErr != nil {
		writeCatalogError(writer, request, includeYankedErr)
		return
	}
	pageSize, err := parsePageSize(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	cursor, err := parseCursor(request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}

	response, repoErr := s.repo.ListVersions(request.Context(), catalog.VersionQuery{
		PackageName:       packageName,
		IncludeDeprecated: includeDeprecated,
		IncludeYanked:     includeYanked,
		PageSize:          pageSize,
		Cursor:            cursor,
	})
	if repoErr != nil {
		writeCatalogError(writer, request, repoErr)
		return
	}

	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleVersionDetail(writer http.ResponseWriter, request *http.Request, packageName string, version string) {
	decodedVersion, decodeErr := decodeVersion(version)
	if decodeErr != nil {
		writeCatalogError(writer, request, decodeErr)
		return
	}

	response, err := s.repo.GetVersion(request.Context(), packageName, decodedVersion)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleMockRegistry(writer http.ResponseWriter, request *http.Request) {
	if s.mockRegistry == nil {
		http.NotFound(writer, request)
		return
	}

	remainder := strings.TrimPrefix(request.URL.EscapedPath(), "/mock-npm/")
	if strings.Contains(remainder, "/-/") {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		packagePart, filePart, ok := strings.Cut(remainder, "/-/")
		if !ok {
			http.NotFound(writer, request)
			return
		}
		packageName := "@" + packagePart
		version := strings.TrimSuffix(strings.TrimPrefix(filePart, packageBasename(packageName)+"-"), ".tgz")
		tarball, found := s.mockRegistry.Tarball(packageName, version)
		if !found {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/octet-stream")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(tarball)
		return
	}

	packageName, err := url.PathUnescape(remainder)
	if err != nil {
		http.NotFound(writer, request)
		return
	}

	switch request.Method {
	case http.MethodGet:
		payload, found := s.mockRegistry.PackageMetadata(mockRegistryBaseURL(request), packageName)
		if !found {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(payload)
	case http.MethodPut:
		body, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			http.Error(writer, "failed to read request body", http.StatusBadRequest)
			return
		}
		if !s.mockRegistry.PublishPackage(packageName, body) {
			http.Error(writer, "invalid publish payload", http.StatusBadRequest)
			return
		}
		writer.WriteHeader(http.StatusCreated)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleInstallTarget(writer http.ResponseWriter, request *http.Request, packageName string) {
	response, err := s.repo.ResolveInstallTarget(request.Context(), packageName, request.URL.Query().Get("version"))
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	response = s.publicInstallTarget(response)
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) publicInstallTarget(response catalog.InstallResolution) catalog.InstallResolution {
	if s.publicNPMRegistryURL == "" {
		return response
	}

	internalRegistry := strings.TrimRight(strings.TrimSpace(response.NPMRegistry), "/")
	response.NPMRegistry = s.publicNPMRegistryURL
	response.TarballURL = rewriteRegistryBaseURL(response.TarballURL, internalRegistry, s.publicNPMRegistryURL)
	return response
}

func rewriteRegistryBaseURL(rawURL string, fromBase string, toBase string) string {
	trimmedURL := strings.TrimSpace(rawURL)
	normalizedFrom := strings.TrimRight(strings.TrimSpace(fromBase), "/")
	normalizedTo := strings.TrimRight(strings.TrimSpace(toBase), "/")

	switch {
	case trimmedURL == "":
		return rawURL
	case normalizedFrom == "":
		return trimmedURL
	case trimmedURL == normalizedFrom:
		return normalizedTo
	case strings.HasPrefix(trimmedURL, normalizedFrom+"/"):
		return normalizedTo + strings.TrimPrefix(trimmedURL, normalizedFrom)
	default:
		return trimmedURL
	}
}

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
	if err := decodeJSONBody(request, &payload); err != nil {
		s.recordMutationAudit(request, "packaging.cleanup", "", "", err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		writeCatalogError(writer, request, err)
		return
	}

	if strings.TrimSpace(payload.WorkspacePath) == "" {
		err := catalog.NewError(422, "invalid_request", "workspacePath is required.", nil)
		s.recordMutationAudit(request, "packaging.cleanup", "", "", err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		writeCatalogError(writer, request, err)
		return
	}

	if err := s.packaging.Cleanup(payload.WorkspacePath); err != nil {
		s.recordMutationAudit(request, "packaging.cleanup", "", "", statusFromError(err), "failed", map[string]any{
			"errorCode": errorCodeFromError(err),
		})
		writeCatalogError(writer, request, err)
		return
	}

	s.recordMutationAudit(request, "packaging.cleanup", "", "", http.StatusOK, "succeeded", map[string]any{
		"workspacePath": payload.WorkspacePath,
	})
	writeJSON(writer, http.StatusOK, packaging.CleanupResponse{
		WorkspacePath: payload.WorkspacePath,
		Removed:       true,
	})
}

func (s *Server) handlePackagingBundleDownload(writer http.ResponseWriter, request *http.Request) {
	if s.packaging == nil {
		http.NotFound(writer, request)
		return
	}

	workspacePath := strings.TrimSpace(request.URL.Query().Get("workspacePath"))
	packageName := strings.TrimSpace(request.URL.Query().Get("packageName"))
	version := strings.TrimSpace(request.URL.Query().Get("version"))
	if workspacePath == "" || packageName == "" || version == "" {
		err := catalog.NewError(422, "invalid_request", "workspacePath, packageName, and version are required.", nil)
		s.recordMutationAudit(request, "packaging.bundle.download", packageName, version, err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		writeCatalogError(writer, request, err)
		return
	}
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}

	bundlePath, err := s.packaging.BundlePath(workspacePath, packaging.Target{
		PackageName: packageName,
		Version:     version,
	})
	if err != nil {
		s.recordMutationAudit(request, "packaging.bundle.download", packageName, version, statusFromError(err), "failed", map[string]any{
			"errorCode": errorCodeFromError(err),
		})
		writeCatalogError(writer, request, err)
		return
	}

	content, readErr := os.ReadFile(bundlePath)
	if readErr != nil {
		err := catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{
			"workspacePath": workspacePath,
			"packageName":   packageName,
			"version":       version,
		})
		s.recordMutationAudit(request, "packaging.bundle.download", packageName, version, err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		writeCatalogError(writer, request, err)
		return
	}

	s.recordMutationAudit(request, "packaging.bundle.download", packageName, version, http.StatusOK, "succeeded", nil)
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(content)
}

func (s *Server) handleReleaseRecord(writer http.ResponseWriter, request *http.Request, packageName string, versionPath string) {
	version, err := decodeVersion(versionPath)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}

	input, decodeErr := decodeOptionalReleaseRecordInput(request)
	if decodeErr != nil {
		writeCatalogError(writer, request, decodeErr)
		return
	}

	response, repoErr := s.repo.CreateReleaseRecord(request.Context(), packageName, version, input)
	if repoErr != nil {
		s.recordMutationAudit(request, "release.record", packageName, version, statusFromError(repoErr), "failed", map[string]any{
			"errorCode": errorCodeFromError(repoErr),
		})
		writeCatalogError(writer, request, repoErr)
		return
	}

	status := http.StatusOK
	if response.Created {
		status = http.StatusCreated
	}
	s.recordMutationAudit(request, "release.record", packageName, version, status, "succeeded", map[string]any{
		"created":          response.Created,
		"idempotentReplay": response.IdempotentReplay,
	})
	writeJSON(writer, status, response)
}

func (s *Server) handleSkillGovernanceChange(writer http.ResponseWriter, request *http.Request, packageName string) {
	var payload catalog.SkillGovernanceChangeInput
	if err := decodeJSONBody(request, &payload); err != nil {
		writeCatalogError(writer, request, err)
		return
	}

	response, repoErr := s.repo.ApplySkillGovernanceChange(request.Context(), packageName, payload, auth.ActorFromRequest(request))
	if repoErr != nil {
		s.recordMutationAudit(request, "governance.skill", packageName, "", statusFromError(repoErr), "failed", map[string]any{
			"action":    payload.Action,
			"errorCode": errorCodeFromError(repoErr),
		})
		writeCatalogError(writer, request, repoErr)
		return
	}

	s.recordMutationAudit(request, "governance.skill", packageName, "", http.StatusCreated, "succeeded", map[string]any{
		"action": payload.Action,
	})
	writeJSON(writer, http.StatusCreated, response)
}

func (s *Server) handleVersionGovernanceChange(writer http.ResponseWriter, request *http.Request, packageName string, versionPath string) {
	version, err := decodeVersion(versionPath)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}

	var payload catalog.VersionGovernanceChangeInput
	if decodeErr := decodeJSONBody(request, &payload); decodeErr != nil {
		writeCatalogError(writer, request, decodeErr)
		return
	}

	response, repoErr := s.repo.ApplyVersionGovernanceChange(request.Context(), packageName, version, payload, auth.ActorFromRequest(request))
	if repoErr != nil {
		s.recordMutationAudit(request, "governance.version", packageName, version, statusFromError(repoErr), "failed", map[string]any{
			"action":    payload.Action,
			"errorCode": errorCodeFromError(repoErr),
		})
		writeCatalogError(writer, request, repoErr)
		return
	}

	s.recordMutationAudit(request, "governance.version", packageName, version, http.StatusCreated, "succeeded", map[string]any{
		"action": payload.Action,
	})
	writeJSON(writer, http.StatusCreated, response)
}

func splitSkillPath(targetURL *url.URL) (string, []string, *catalog.Error) {
	path := targetURL.EscapedPath()
	const prefix = "/api/v1/skills/"
	if !strings.HasPrefix(path, prefix) {
		return "", nil, catalog.NewError(400, "invalid_request", "path is invalid.", nil)
	}

	remainder := strings.TrimPrefix(path, prefix)
	if remainder == "" {
		return "", nil, catalog.NewError(400, "invalid_request", "packageName path segment is required.", nil)
	}

	parts := strings.Split(remainder, "/")
	packageName, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", nil, catalog.NewError(400, "invalid_request", "packageName path is not valid URL encoding.", map[string]any{"packageName": parts[0]})
	}
	return packageName, parts[1:], nil
}

func decodeVersion(raw string) (string, *catalog.Error) {
	version, err := url.PathUnescape(raw)
	if err != nil {
		return "", catalog.NewError(400, "invalid_request", "version path is not valid URL encoding.", map[string]any{"version": raw})
	}
	if !isExactVersionString(version) {
		return "", catalog.NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
	}
	return version, nil
}

func parsePageSize(values url.Values) (int, *catalog.Error) {
	raw := strings.TrimSpace(values.Get("pageSize"))
	if raw == "" {
		return 20, nil
	}
	pageSize, err := strconv.Atoi(raw)
	if err != nil {
		return 0, catalog.NewError(400, "invalid_request", "pageSize must be an integer.", map[string]any{"pageSize": raw})
	}
	if pageSize <= 0 {
		return 0, catalog.NewError(400, "invalid_request", "pageSize must be greater than or equal to 1.", map[string]any{"pageSize": raw})
	}
	return pageSize, nil
}

func parseCursor(values url.Values) (int, *catalog.Error) {
	raw := strings.TrimSpace(values.Get("cursor"))
	if raw == "" {
		return 0, nil
	}
	cursor, err := parseCursorToken(raw)
	if err != nil {
		return 0, catalog.NewError(400, "invalid_request", "cursor must be a valid cursor token.", map[string]any{"cursor": raw})
	}
	return cursor, nil
}

func parseCursorToken(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "cursor_") {
		value = strings.TrimPrefix(value, "cursor_")
	}
	return strconv.Atoi(value)
}

func parseBooleanQueryParam(raw string, field string, defaultValue bool) (bool, *catalog.Error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultValue, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes":
		return true, nil
	case "0", "false", "no":
		return false, nil
	default:
		return false, catalog.NewError(400, "invalid_request", field+" must be a boolean.", map[string]any{field: value})
	}
}

func parseSkillsSort(raw string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "", "relevance", "updated_at_desc", "package_name_asc":
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "sort is invalid.", map[string]any{"sort": value})
	}
}

func parseSourceTypeFilter(raw string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "", string(catalog.SourceTypeNative), string(catalog.SourceTypeImported), string(catalog.SourceTypePlatformCreated):
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "source_type is invalid.", map[string]any{"source_type": value})
	}
}

func parseGovernanceTargetTypeFilter(raw string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "", string(catalog.GovernanceTargetSkill), string(catalog.GovernanceTargetVersion):
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "target_type is invalid.", map[string]any{"target_type": value})
	}
}

func parseAuthDecisionFilter(raw string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "", string(observability.AuthDecisionAllowed), string(observability.AuthDecisionDenied):
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "auth_decision must be one of: allowed, denied.", map[string]any{"auth_decision": value})
	}
}

func parseExactVersionQueryParam(raw string, field string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if !isExactVersionString(value) {
		return "", catalog.NewError(400, "invalid_request", field+" must be an exact semver string.", map[string]any{field: value})
	}
	return value, nil
}

func isExactVersionString(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func decodeJSONBody(request *http.Request, target any) *catalog.Error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		if err == io.EOF {
			return catalog.NewError(422, "invalid_request", "request body is required.", nil)
		}
		return catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.NewError(422, "invalid_request", "request body must contain a single JSON object.", nil)
	}

	return nil
}

func decodeJSONBytes(raw []byte, target any) *catalog.Error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		if err == io.EOF {
			return catalog.NewError(422, "invalid_request", "request body is required.", nil)
		}
		return catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.NewError(422, "invalid_request", "request body must contain a single JSON object.", nil)
	}

	return nil
}

func decodeOptionalReleaseRecordInput(request *http.Request) (catalog.ReleaseRecordInput, *catalog.Error) {
	if request.Body == nil {
		return catalog.ReleaseRecordInput{}, nil
	}

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	var payload catalog.ReleaseRecordInput
	if err := decoder.Decode(&payload); err != nil {
		if err == io.EOF {
			return catalog.ReleaseRecordInput{}, nil
		}
		return catalog.ReleaseRecordInput{}, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.ReleaseRecordInput{}, catalog.NewError(422, "invalid_request", "request body must contain a single JSON object.", nil)
	}

	return payload, nil
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func writeJSONBytes(writer http.ResponseWriter, status int, payload []byte) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write(payload)
}

func writeCatalogError(writer http.ResponseWriter, request *http.Request, err error) {
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		catalogErr = catalog.NewError(500, "internal_error", "Internal server error.", nil)
	}

	requestID, _ := request.Context().Value(requestIDContextKey).(string)
	writeJSON(writer, catalogErr.Status, map[string]any{
		"error": map[string]any{
			"code":      catalogErr.Code,
			"message":   catalogErr.Message,
			"details":   catalogErr.Details,
			"requestId": requestID,
		},
	})
}

func writeAuthorizationError(writer http.ResponseWriter, request *http.Request, err *auth.AuthorizationError) {
	writeCatalogError(writer, request, err.CatalogError())
}

func (s *Server) authorizeActionAccess(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool {
	current := auth.PrincipalFromContext(request.Context())
	observability.SetObservedAction(writer, string(action))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	if authErr := s.authorizer.AuthorizeAction(current, action); authErr != nil {
		s.denyAuthorization(writer, request, current, authErr, packageName, version)
		return false
	}
	return true
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func traceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDContextKey).(string)
	return traceID
}

func packageBasename(packageName string) string {
	parts := strings.Split(packageName, "/")
	return parts[len(parts)-1]
}

func mockRegistryBaseURL(request *http.Request) string {
	scheme := request.URL.Scheme
	if scheme == "" {
		scheme = "http"
	}
	host := request.Host
	if host == "" {
		host = request.URL.Host
	}
	return scheme + "://" + host + "/mock-npm"
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
		writeCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	principal := auth.PrincipalFromContext(request.Context())
	payload, err := decode(request.Context(), principal, rawBody)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	if packageName := payload.GetTarget().PackageName; packageName != "" {
		if !s.authorizePackageAccess(writer, request, packageName) {
			return
		}
	}
	response, status, err := execute(request.Context(), principal, payload)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, status, response)
}

func hashRequestPayload(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Server) recordMutationAudit(request *http.Request, action string, packageName string, version string, status int, outcome string, details map[string]any) {
	if err := s.auditRecorder.RecordMutation(request, action, packageName, version, status, outcome, details); err != nil {
		s.logAuditWriteFailure(request, action, err)
	}
}

func (s *Server) recordAuthenticationDeniedAudit(request *http.Request) {
	if err := s.auditRecorder.RecordAuthenticationDenied(request); err != nil {
		s.logAuditWriteFailure(request, "auth.denied", err)
	}
}

func (s *Server) logAuditWriteFailure(request *http.Request, action string, err error) {
	if err == nil {
		return
	}
	s.logger.Printf(
		`{"event":"audit_write_failed","requestId":%q,"traceId":%q,"action":%q,"authDecision":%q,"error":%q}`,
		requestIDFromContext(request.Context()),
		traceIDFromContext(request.Context()),
		action,
		observability.AuthDecisionAllowed,
		err.Error(),
	)
}

func (s *Server) logRequest(request *http.Request, recorder *observability.StatusRecorder, startedAt time.Time) {
	fields := map[string]any{
		"event":         "http_request",
		"requestId":     requestIDFromContext(request.Context()),
		"traceId":       traceIDFromContext(request.Context()),
		"method":        request.Method,
		"path":          request.URL.EscapedPath(),
		"status":        recorder.Status,
		"durationMs":    time.Since(startedAt).Milliseconds(),
		"role":          recorder.ObservedPrincipal.Role,
		"actor":         recorder.ObservedPrincipal.Actor,
		"authMechanism": recorder.ObservedPrincipal.AuthMechanism,
		"action":        recorder.ObservedAction,
		"authDecision":  recorder.ObservedAuthDecision,
		"outcome":       recorder.Outcome(),
	}
	if payload, err := json.Marshal(fields); err == nil {
		s.logger.Print(string(payload))
	}
}

func statusFromError(err error) int {
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		return http.StatusInternalServerError
	}
	return catalogErr.Status
}

func errorCodeFromError(err error) string {
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		return "internal_error"
	}
	return catalogErr.Code
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func (s *Server) authorizePackageAccess(writer http.ResponseWriter, request *http.Request, packageName string) bool {
	current := auth.PrincipalFromContext(request.Context())
	if authErr := s.authorizer.AuthorizePackage(current, packageName); authErr == nil {
		return true
	} else {
		s.denyAuthorization(writer, request, current, authErr, packageName, "")
		return false
	}
}

func (s *Server) denyAuthorization(writer http.ResponseWriter, request *http.Request, current auth.Principal, authErr *auth.AuthorizationError, packageName string, version string) {
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionDenied)
	observability.SetObservedOutcome(writer, authErr.Code)
	if err := s.auditRecorder.RecordAuthorizationDenied(request, current, authErr.Code, authErr.Status, packageName, version, authErr.Details); err != nil {
		s.logAuditWriteFailure(request, "auth.denied", err)
	}
	writeAuthorizationError(writer, request, authErr)
}
