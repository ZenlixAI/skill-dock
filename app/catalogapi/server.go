package catalogapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/audit"
	"skill_registry/app/catalogapi/common"
	catalogmiddleware "skill_registry/app/catalogapi/middleware"
	routerpkg "skill_registry/app/catalogapi/router"
	"skill_registry/app/catalogapi/system"
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
	publicNPMRegistryURL  string
	packaging             *packaging.Service
	auditRecorder         *audit.Recorder
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
	auditRecorder := audit.NewRecorder(config.Audit)
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
	server.auditQueryModule = audit.NewQueryModule(config.Audit, authorizer)
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
	server.systemModule = system.NewModule(
		system.NamedDependency{Name: "repository", Pinger: system.DependencyPinger(config.Repository)},
		system.NamedDependency{Name: "audit", Pinger: system.DependencyPinger(config.Audit)},
		system.NamedDependency{Name: "idempotency", Pinger: system.DependencyPinger(config.Idempotency)},
	)
	return server
}

func (s *Server) authorizePackageAccess(writer http.ResponseWriter, request *http.Request, packageName string) bool {
	return s.accessWorkflow.authorizePackage(writer, request, packageName)
}

func (s *Server) authorizeActionAccess(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool {
	return s.accessWorkflow.authorizeAction(writer, request, action, packageName, version)
}

func (s *Server) serveCachedReadJSON(writer http.ResponseWriter, request *http.Request, ttl time.Duration, load func() (any, *catalog.Error)) {
	release, limitErr := s.readLimiter.acquire()
	if limitErr != nil {
		common.WriteCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	cacheKey := s.readCacheKey(request, auth.PrincipalFromContext(request.Context()))
	if cached, ok := s.readCache.lookup(cacheKey); ok {
		common.WriteJSONBytes(writer, http.StatusOK, cached)
		return
	}

	response, err := load()
	if err != nil {
		common.WriteCatalogError(writer, request, err)
		return
	}

	raw, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		common.WriteCatalogError(writer, request, marshalErr)
		return
	}
	s.readCache.store(cacheKey, raw, ttl)
	common.WriteJSONBytes(writer, http.StatusOK, raw)
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
		common.RequestIDFromContext(request.Context()),
		common.TraceIDFromContext(request.Context()),
		action,
		observability.AuthDecisionAllowed,
		err.Error(),
	)
}

func (s *Server) routerConfig() routerpkg.Config {
	return routerpkg.Config{
		ServesReadRoutes:  s.mode.servesReadRoutes(),
		ServesWriteRoutes: s.mode.servesWriteRoutes(),
		Middleware: []gin.HandlerFunc{
			catalogmiddleware.WithCORS(),
			catalogmiddleware.WithRequestMetadata(&s.requestID),
			catalogmiddleware.WithObservability(s.metrics, s.logger),
			catalogmiddleware.WithAuthentication(s.authenticateRequest),
		},
		Healthz:           s.ginHandler(http.HandlerFunc(s.handleHealthz)),
		Readyz:            s.ginHandler(http.HandlerFunc(s.handleReadyz)),
		Metrics:           s.ginHandler(http.HandlerFunc(s.handleMetrics)),
		Session:           s.ginHandler(http.HandlerFunc(s.handleSession)),
		SessionLogin:      s.ginHandler(http.HandlerFunc(s.handleSessionLogin)),
		SessionLogout:     s.ginHandler(http.HandlerFunc(s.handleSessionLogout)),
		AuditEvents:       s.protectedGinHandler(auth.ActionReadAuditEvents, http.HandlerFunc(s.handleAuditEvents)),
		Skills:            s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleSkills)),
		SkillsSearch:      s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleSkills)),
		CuratedSkills:     s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleCuratedSkills)),
		GovernanceChanges: s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleGovernanceChanges)),
		SkillDetail:       s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleSkillDetailRoute), "package"),
		VersionList:       s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleVersionListRoute), "package"),
		VersionDetail:     s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleVersionDetailRoute), "package", "version"),
		InstallTarget:     s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleInstallTargetRoute), "package"),
		SkillGovernance:   s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleSkillGovernanceRoute), "package"),
		ReleaseRecord:     s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleReleaseRecordRoute), "package", "version"),
		VersionGovernance: s.protectedGinHandler(auth.ActionReadCatalog, http.HandlerFunc(s.handleVersionGovernanceRoute), "package", "version"),
		PackagingPrepare:  s.protectedGinHandler(auth.ActionPackagingPrepare, http.HandlerFunc(s.handlePackagingPrepare)),
		PackagingValidate: s.protectedGinHandler(auth.ActionPackagingValidate, http.HandlerFunc(s.handlePackagingValidate)),
		PackagingPack:     s.protectedGinHandler(auth.ActionPackagingPack, http.HandlerFunc(s.handlePackagingPack)),
		PackagingPublish:  s.protectedGinHandler(auth.ActionPackagingPublish, http.HandlerFunc(s.handlePackagingPublish)),
		PackagingCleanup:  s.protectedGinHandler(auth.ActionPackagingPack, http.HandlerFunc(s.handlePackagingCleanup)),
		PackagingBundle:   s.protectedGinHandler(auth.ActionPackagingPack, http.HandlerFunc(s.handlePackagingBundleDownload)),
	}
}

func (s *Server) ginHandler(handler http.Handler, pathParams ...string) gin.HandlerFunc {
	return func(context *gin.Context) {
		request := context.Request
		for _, key := range pathParams {
			value := context.Param(key)
			if decoded, err := url.PathUnescape(value); err == nil {
				value = decoded
			}
			request.SetPathValue(key, value)
		}
		handler.ServeHTTP(context.Writer, request)
	}
}

func (s *Server) protectedGinHandler(action auth.Action, handler http.Handler, pathParams ...string) gin.HandlerFunc {
	return catalogmiddleware.RequireAction(
		s.ginHandler(handler, pathParams...),
		action,
		s.accessWorkflow.authorizeAction,
		s.recordAuthenticationDeniedAudit,
	)
}

func (s *Server) withCanonicalRoutes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if target, ok := s.canonicalRouteTarget(request); ok {
			startedAt := time.Now()
			request = s.prepareCanonicalRedirectRequest(writer, request)
			http.Redirect(writer, request, target, http.StatusTemporaryRedirect)
			s.recordCanonicalRedirect(request, http.StatusTemporaryRedirect, time.Since(startedAt))
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) prepareCanonicalRedirectRequest(writer http.ResponseWriter, request *http.Request) *http.Request {
	request = s.withCanonicalRequestMetadata(writer, request)
	request = common.EnsureObservedState(request)
	s.applyCanonicalCORSHeaders(writer, request)
	return request
}

func (s *Server) withCanonicalRequestMetadata(writer http.ResponseWriter, request *http.Request) *http.Request {
	if request == nil {
		return nil
	}

	requestID := strings.TrimSpace(common.RequestIDFromContext(request.Context()))
	if requestID == "" {
		requestID = "req_" + strconv.FormatUint(s.requestID.Add(1), 10)
	}

	traceID := strings.TrimSpace(common.TraceIDFromContext(request.Context()))
	if traceID == "" {
		traceID = strings.TrimSpace(request.Header.Get("X-Trace-Id"))
		if traceID == "" {
			traceID = "trace_" + strings.TrimPrefix(requestID, "req_")
		}
	}

	writer.Header().Set("X-Request-Id", requestID)
	writer.Header().Set("X-Trace-Id", traceID)
	return request.WithContext(common.ContextWithRequestMetadata(request.Context(), requestID, traceID))
}

func (s *Server) applyCanonicalCORSHeaders(writer http.ResponseWriter, request *http.Request) {
	if request == nil {
		return
	}
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" || !isLocalOperatorOrigin(origin) {
		return
	}
	writer.Header().Set("Access-Control-Allow-Origin", origin)
	writer.Header().Set("Vary", "Origin")
	writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Actor, X-Role, X-Trace-Id")
	writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
	writer.Header().Set("Access-Control-Allow-Credentials", "true")
	writer.Header().Set("Access-Control-Max-Age", "600")
}

func (s *Server) recordCanonicalRedirect(request *http.Request, status int, duration time.Duration) {
	path := ""
	method := ""
	var state *common.ObservedState
	if request != nil {
		method = request.Method
		if request.URL != nil {
			path = request.URL.EscapedPath()
			if path == "" {
				path = request.URL.Path
			}
		}
		state = common.ObservedStateFromContext(request.Context())
	}
	if state == nil {
		state = &common.ObservedState{}
	}
	outcome := common.ObservedOutcome(status, state)

	if s.metrics != nil {
		s.metrics.RecordRequest(method, path, status, duration, state.Action, state.AuthDecision, outcome)
	}
	if s.logger != nil {
		fields := map[string]any{
			"event":         "http_request",
			"requestId":     requestMetadataValue(request, common.RequestIDFromContext),
			"traceId":       requestMetadataValue(request, common.TraceIDFromContext),
			"method":        method,
			"path":          path,
			"status":        status,
			"durationMs":    duration.Milliseconds(),
			"role":          state.Principal.Role,
			"actor":         state.Principal.Actor,
			"authMechanism": state.Principal.AuthMechanism,
			"action":        state.Action,
			"authDecision":  state.AuthDecision,
			"outcome":       outcome,
		}
		if payload, err := json.Marshal(fields); err == nil {
			s.logger.Print(string(payload))
		}
	}
}

func requestMetadataValue(request *http.Request, lookup func(context.Context) string) string {
	if request == nil {
		return ""
	}
	return lookup(request.Context())
}

func (s *Server) canonicalRouteTarget(request *http.Request) (string, bool) {
	if request == nil || request.URL == nil || request.Method == http.MethodConnect {
		return "", false
	}

	escapedPath := request.URL.EscapedPath()
	if escapedPath == "" {
		escapedPath = request.URL.Path
	}
	cleanedPath := cleanCanonicalPath(escapedPath)
	if cleanedPath != escapedPath {
		return canonicalRedirectTarget(cleanedPath, request.URL.RawQuery), true
	}
	return "", false
}

func cleanCanonicalPath(rawPath string) string {
	if rawPath == "" {
		return "/"
	}
	if rawPath[0] != '/' {
		rawPath = "/" + rawPath
	}
	cleanedPath := path.Clean(rawPath)
	if rawPath[len(rawPath)-1] == '/' && cleanedPath != "/" {
		if len(rawPath) == len(cleanedPath)+1 && strings.HasPrefix(rawPath, cleanedPath) {
			cleanedPath = rawPath
		} else {
			cleanedPath += "/"
		}
	}
	return cleanedPath
}

func canonicalRedirectTarget(path string, rawQuery string) string {
	if strings.TrimSpace(rawQuery) == "" {
		return path
	}
	return path + "?" + rawQuery
}

func isLocalOperatorOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "127.0.0.1" || host == "localhost"
}
