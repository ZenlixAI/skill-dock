package catalogapi

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/audit"
	"skill_registry/app/catalogapi/common"
	sessionhandlers "skill_registry/app/catalogapi/handlers/session"
	systemhandlers "skill_registry/app/catalogapi/handlers/system"
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
	systemHTTPHandlers := systemhandlers.New(s.systemModule.Readyz, s.metrics.RenderPrometheus)
	sessionHTTPHandlers := sessionhandlers.New(sessionhandlers.Handlers{
		Authenticate:    s.authenticateRequest,
		AuthorizeAction: s.accessWorkflow.authorizeAction,
		CurrentFunc: func(principal auth.Principal) any {
			if s.sessionModule == nil {
				return nil
			}
			return s.sessionModule.current(principal)
		},
		LoginFunc: func(request *http.Request, username string, password string) (auth.Principal, *http.Cookie, bool) {
			if s.sessionModule == nil {
				return auth.Principal{}, nil, false
			}
			return s.sessionModule.login(request, username, password)
		},
		LogoutCookie: func(request *http.Request) *http.Cookie {
			if s.sessionModule == nil {
				return nil
			}
			return s.sessionModule.logoutCookie(request)
		},
		Enabled: func() bool {
			return s.sessionModule != nil && s.sessionModule.enabled()
		},
	})

	return routerpkg.Config{
		ServesReadRoutes:  s.mode.servesReadRoutes(),
		ServesWriteRoutes: s.mode.servesWriteRoutes(),
		Middleware: []gin.HandlerFunc{
			catalogmiddleware.WithCORS(),
			catalogmiddleware.WithRequestMetadata(&s.requestID),
			catalogmiddleware.WithObservability(s.metrics, s.logger),
			catalogmiddleware.WithCanonicalRedirect(),
			catalogmiddleware.WithAuthentication(s.authenticateRequest),
		},
		Healthz:           systemHTTPHandlers.Healthz,
		Readyz:            systemHTTPHandlers.Readyz,
		Metrics:           systemHTTPHandlers.Metrics,
		Session:           sessionHTTPHandlers.Current,
		SessionLogin:      sessionHTTPHandlers.Login,
		SessionLogout:     sessionHTTPHandlers.Logout,
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
