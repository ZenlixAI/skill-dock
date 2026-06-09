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
	"path/filepath"
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
	Repository                      catalog.Repository
	Mode                            ServerMode
	ReadListCacheTTL                time.Duration
	ReadDetailCacheTTL              time.Duration
	ReadMaxInFlight                 int
	WriteMaxInFlight                int
	ReaderToken                     string
	AuthoringToken                  string
	PublisherToken                  string
	NPMRegistryURL                  string
	NPMRegistryPublishAuthorization string
	NPMRegistryPublishClient        *http.Client
	PublicNPMRegistryURL            string
	AuthTokens                      map[string]auth.TokenConfig
	SessionSecret                   string
	SessionUsers                    map[string]auth.SessionUserConfig
	MockRegistry                    *artifactadapter.MockRegistry
	DisableMockRegistry             bool
	Packaging                       *packaging.Service
	Idempotency                     persist.IdempotencyStore
	Audit                           persist.AuditStore
	Logger                          *log.Logger
}

type Server struct {
	repo                  catalog.Repository
	authorizer            *auth.Authorizer
	sessionManager        *auth.SessionManager
	accessWorkflow        accessWorkflowHandler
	catalogReadModule     catalogReadWorkflow
	readAPIModule         readAPIWorkflow
	auditQueryModule      auditQueryWorkflow
	catalogMutationModule catalogMutationWorkflow
	governanceModule      *governanceModule
	packagingModule       *packagingModule
	packagingAccessModule packagingAccessWorkflow
	sessionModule         sessionWorkflow
	systemModule          systemWorkflow
	mockRegistryModule    mockRegistryWorkflow
	publicNPMRegistryURL  string
	disableMockRegistry   bool
	packaging             *packaging.Service
	auditRecorder         *AuditRecorder
	logger                *log.Logger
	metrics               *observability.Collector
	requestID             atomic.Uint64
	mode                  ServerMode
	readListCacheTTL      time.Duration
	readDetailCacheTTL    time.Duration
	readCache             *readCache
	readLimiter           *requestLimiter
	writeLimiter          *requestLimiter
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
	mode := normalizeServerMode(config.Mode)
	readListCacheTTL := config.ReadListCacheTTL
	if readListCacheTTL <= 0 {
		readListCacheTTL = 2 * time.Second
	}
	readDetailCacheTTL := config.ReadDetailCacheTTL
	if readDetailCacheTTL <= 0 {
		readDetailCacheTTL = 5 * time.Second
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
		config.Packaging.ConfigurePublishWorkflow(
			config.Repository,
			artifactadapter.NewNPMPublisher(
				strings.TrimRight(strings.TrimSpace(config.NPMRegistryURL), "/"),
				strings.TrimSpace(config.NPMRegistryPublishAuthorization),
				config.NPMRegistryPublishClient,
			),
		)
		packagingSvc = config.Packaging
	}

	server := &Server{
		repo:                 config.Repository,
		authorizer:           authorizer,
		sessionManager:       sessionManager,
		publicNPMRegistryURL: strings.TrimRight(strings.TrimSpace(config.PublicNPMRegistryURL), "/"),
		disableMockRegistry:  config.DisableMockRegistry,
		packaging:            config.Packaging,
		auditRecorder:        auditRecorder,
		logger:               logger,
		metrics:              observability.NewCollector(),
		mode:                 mode,
		readListCacheTTL:     readListCacheTTL,
		readDetailCacheTTL:   readDetailCacheTTL,
		readCache:            newReadCache(),
		readLimiter:          newRequestLimiter(config.ReadMaxInFlight),
		writeLimiter:         newRequestLimiter(config.WriteMaxInFlight),
	}
	server.catalogReadModule = &catalogReadModule{
		repo: config.Repository,
	}
	server.readAPIModule = &readAPIModule{
		catalogReadModule: server.catalogReadModule,
	}
	server.accessWorkflow = &accessWorkflow{
		authorizer: authorizer,
		audit:      auditRecorder,
		logger:     logger,
	}
	server.auditQueryModule = &auditQueryModule{
		audit:      config.Audit,
		authorizer: authorizer,
	}
	server.catalogMutationModule = &catalogMutationModule{
		repo:  config.Repository,
		audit: auditRecorder,
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
	server.packagingAccessModule = &packagingAccessModule{
		service: config.Packaging,
		audit:   auditRecorder,
	}
	server.sessionModule = &sessionModule{
		authorizer:     authorizer,
		sessionManager: sessionManager,
	}
	server.systemModule = &systemModule{
		repository: namedReadinessDependency{name: "repository", pinger: dependencyPinger(config.Repository)},
		audit:      namedReadinessDependency{name: "audit", pinger: dependencyPinger(config.Audit)},
		idempotency: namedReadinessDependency{
			name:   "idempotency",
			pinger: dependencyPinger(config.Idempotency),
		},
	}
	server.mockRegistryModule = &mockRegistryModule{
		registry: config.MockRegistry,
	}
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(s.handleHealthz))
	mux.Handle("/readyz", http.HandlerFunc(s.handleReadyz))
	mux.Handle("/metrics", http.HandlerFunc(s.handleMetrics))
	mux.Handle("/api/v1/session", http.HandlerFunc(s.handleSession))
	mux.Handle("/api/v1/session/login", http.HandlerFunc(s.handleSessionLogin))
	mux.Handle("/api/v1/session/logout", http.HandlerFunc(s.handleSessionLogout))
	if s.mode.servesWriteRoutes() {
		mux.Handle("/api/v1/audit-events", s.requireAction(s.handleAuditEvents, auth.ActionReadAuditEvents))
	}
	if s.mode.servesReadRoutes() {
		mux.Handle("/api/v1/skills", s.requireAction(s.handleSkills, auth.ActionReadCatalog))
		mux.Handle("/api/v1/skills/search", s.requireAction(s.handleSkills, auth.ActionReadCatalog))
		mux.Handle("/api/v1/skills/curated", s.requireAction(s.handleCuratedSkills, auth.ActionReadCatalog))
		mux.Handle("/api/v1/governance-changes", s.requireAction(s.handleGovernanceChanges, auth.ActionReadCatalog))
	}
	if s.mode.servesReadRoutes() || s.mode.servesWriteRoutes() {
		mux.Handle("/api/v1/skills/", s.requireAction(s.handleSkillSubtree, auth.ActionReadCatalog))
	}
	if s.mode.servesWriteRoutes() {
		mux.Handle("/api/v1/skill-packaging/prepare", s.requireAction(s.handlePackagingPrepare, auth.ActionPackagingPrepare))
		mux.Handle("/api/v1/skill-packaging/validate", s.requireAction(s.handlePackagingValidate, auth.ActionPackagingValidate))
		mux.Handle("/api/v1/skill-packaging/pack", s.requireAction(s.handlePackagingPack, auth.ActionPackagingPack))
		mux.Handle("/api/v1/skill-packaging/publish", s.requireAction(s.handlePackagingPublish, auth.ActionPackagingPublish))
		mux.Handle("/api/v1/skill-packaging/cleanup", s.requireAction(s.handlePackagingCleanup, auth.ActionPackagingPack))
		mux.Handle("/api/v1/skill-packaging/bundle", s.requireAction(s.handlePackagingBundleDownload, auth.ActionPackagingPack))
	}
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
		if !s.accessWorkflow.authorizeAction(writer, request, action, "", "") {
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

	s.serveCachedReadJSON(writer, request, s.readListCacheTTL, func() (any, *catalog.Error) {
		return s.readAPIModule.listSkills(request.Context(), auth.PrincipalFromContext(request.Context()), request.URL.Query())
	})
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

	s.serveCachedReadJSON(writer, request, s.readListCacheTTL, func() (any, *catalog.Error) {
		return s.catalogReadModule.listCuratedSkills(request.Context(), auth.PrincipalFromContext(request.Context()), pageSize, cursor)
	})
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
	payload, status := s.systemModule.readyz(request.Context())
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
	packageName := request.URL.Query().Get("package_name")
	if packageName != "" && !s.authorizePackageAccess(writer, request, packageName) {
		return
	}

	s.serveCachedReadJSON(writer, request, s.readListCacheTTL, func() (any, *catalog.Error) {
		return s.auditQueryModule.listEvents(request.Context(), auth.PrincipalFromContext(request.Context()), request.URL.Query())
	})
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

	s.serveCachedReadJSON(writer, request, s.readListCacheTTL, func() (any, *catalog.Error) {
		return s.governanceModule.listChanges(request.Context(), auth.PrincipalFromContext(request.Context()), catalog.GovernanceQuery{
			PackageName: request.URL.Query().Get("package_name"),
			Version:     versionFilter,
			TargetType:  targetType,
			PageSize:    pageSize,
			Cursor:      cursor,
		})
	})
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
	case len(tail) == 0 && request.Method == http.MethodGet && s.mode.servesReadRoutes():
		s.handleSkillDetail(writer, request, packageName)
	case len(tail) == 1 && tail[0] == "versions" && request.Method == http.MethodGet && s.mode.servesReadRoutes():
		s.handleVersionList(writer, request, packageName)
	case len(tail) == 1 && tail[0] == "install-target" && request.Method == http.MethodGet && s.mode.servesReadRoutes():
		s.handleInstallTarget(writer, request, packageName)
	case len(tail) == 1 && tail[0] == "governance-changes" && request.Method == http.MethodPost && s.mode.servesWriteRoutes():
		if !s.authorizeActionAccess(writer, request, auth.ActionApplySkillGovernance, packageName, "") {
			return
		}
		s.handleSkillGovernanceChange(writer, request, packageName)
	case len(tail) == 2 && tail[0] == "versions" && request.Method == http.MethodGet && s.mode.servesReadRoutes():
		s.handleVersionDetail(writer, request, packageName, tail[1])
	case len(tail) == 3 && tail[0] == "versions" && tail[2] == "release-record" && request.Method == http.MethodPut && s.mode.servesWriteRoutes():
		if !s.authorizeActionAccess(writer, request, auth.ActionCreateReleaseRecord, packageName, tail[1]) {
			return
		}
		s.handleReleaseRecord(writer, request, packageName, tail[1])
	case len(tail) == 3 && tail[0] == "versions" && tail[2] == "governance-changes" && request.Method == http.MethodPost && s.mode.servesWriteRoutes():
		if !s.authorizeActionAccess(writer, request, auth.ActionApplyVersionGovernance, packageName, tail[1]) {
			return
		}
		s.handleVersionGovernanceChange(writer, request, packageName, tail[1])
	default:
		http.NotFound(writer, request)
	}
}

func (s *Server) handleSkillDetail(writer http.ResponseWriter, request *http.Request, packageName string) {
	s.serveCachedReadJSON(writer, request, s.readDetailCacheTTL, func() (any, *catalog.Error) {
		response, err := s.repo.GetSkill(request.Context(), packageName)
		if err != nil {
			return nil, catalogErrorFromError(err)
		}
		return response, nil
	})
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

	s.serveCachedReadJSON(writer, request, s.readDetailCacheTTL, func() (any, *catalog.Error) {
		response, err := s.repo.ListVersions(request.Context(), catalog.VersionQuery{
			PackageName:       packageName,
			IncludeDeprecated: includeDeprecated,
			IncludeYanked:     includeYanked,
			PageSize:          pageSize,
			Cursor:            cursor,
		})
		if err != nil {
			return nil, catalogErrorFromError(err)
		}
		return response, nil
	})
}

func (s *Server) handleVersionDetail(writer http.ResponseWriter, request *http.Request, packageName string, version string) {
	decodedVersion, decodeErr := decodeVersion(version)
	if decodeErr != nil {
		writeCatalogError(writer, request, decodeErr)
		return
	}

	s.serveCachedReadJSON(writer, request, s.readDetailCacheTTL, func() (any, *catalog.Error) {
		response, err := s.repo.GetVersion(request.Context(), packageName, decodedVersion)
		if err != nil {
			return nil, catalogErrorFromError(err)
		}
		return response, nil
	})
}

func (s *Server) handleMockRegistry(writer http.ResponseWriter, request *http.Request) {
	if s.mockRegistryModule == nil {
		http.NotFound(writer, request)
		return
	}
	s.mockRegistryModule.handleRequest(writer, request)
}

func (s *Server) handleInstallTarget(writer http.ResponseWriter, request *http.Request, packageName string) {
	s.serveCachedReadJSON(writer, request, s.readDetailCacheTTL, func() (any, *catalog.Error) {
		response, err := s.repo.ResolveInstallTarget(request.Context(), packageName, request.URL.Query().Get("version"))
		if err != nil {
			return nil, catalogErrorFromError(err)
		}
		return s.publicInstallTarget(response), nil
	})
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

	response, status, err := s.packagingAccessModule.cleanup(request.Context(), auth.PrincipalFromContext(request.Context()), payload)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, status, response)
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
		writeCatalogError(writer, request, err)
		return
	}

	bundleFile, openErr := os.Open(bundlePath)
	if openErr != nil {
		writeCatalogError(writer, request, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil))
		return
	}
	defer bundleFile.Close()

	info, statErr := bundleFile.Stat()
	if statErr != nil || info.IsDir() {
		writeCatalogError(writer, request, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil))
		return
	}

	writer.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(writer, request, filepath.Base(bundlePath), info.ModTime(), bundleFile)
}

func (s *Server) handleReleaseRecord(writer http.ResponseWriter, request *http.Request, packageName string, versionPath string) {
	release, limitErr := s.writeLimiter.acquire()
	if limitErr != nil {
		writeCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		writeCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
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
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, status, response)
}

func (s *Server) handleSkillGovernanceChange(writer http.ResponseWriter, request *http.Request, packageName string) {
	release, limitErr := s.writeLimiter.acquire()
	if limitErr != nil {
		writeCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		writeCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	response, status, err := s.catalogMutationModule.applySkillGovernanceChange(
		request.Context(),
		auth.PrincipalFromContext(request.Context()),
		packageName,
		rawBody,
	)
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, status, response)
}

func (s *Server) handleVersionGovernanceChange(writer http.ResponseWriter, request *http.Request, packageName string, versionPath string) {
	release, limitErr := s.writeLimiter.acquire()
	if limitErr != nil {
		writeCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		writeCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
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
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, status, response)
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
	return s.accessWorkflow.authorizeAction(writer, request, action, packageName, version)
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func traceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDContextKey).(string)
	return traceID
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
	release, limitErr := s.writeLimiter.acquire()
	if limitErr != nil {
		writeCatalogError(writer, request, limitErr)
		return
	}
	defer release()
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

func (s *Server) serveCachedReadJSON(writer http.ResponseWriter, request *http.Request, ttl time.Duration, load func() (any, *catalog.Error)) {
	release, limitErr := s.readLimiter.acquire()
	if limitErr != nil {
		writeCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	cacheKey := s.readCacheKey(request, auth.PrincipalFromContext(request.Context()))
	if cached, ok := s.readCache.lookup(cacheKey); ok {
		writeJSONBytes(writer, http.StatusOK, cached)
		return
	}

	response, err := load()
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}

	raw, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		writeCatalogError(writer, request, marshalErr)
		return
	}
	s.readCache.store(cacheKey, raw, ttl)
	writeJSONBytes(writer, http.StatusOK, raw)
}

func (s *Server) readCacheKey(request *http.Request, principal auth.Principal) string {
	if request == nil || request.URL == nil {
		return ""
	}
	return strings.Join([]string{
		request.Method,
		request.URL.EscapedPath(),
		request.URL.Query().Encode(),
		principal.Role,
		strings.Join(principal.AllowedPackages, ","),
	}, "\x00")
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
	return s.accessWorkflow.authorizePackage(writer, request, packageName)
}

func (s *Server) denyAuthorization(writer http.ResponseWriter, request *http.Request, current auth.Principal, authErr *auth.AuthorizationError, packageName string, version string) {
	s.accessWorkflow.deny(writer, request, current, authErr, packageName, version)
}
