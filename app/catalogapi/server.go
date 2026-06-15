package catalogapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/catalogapi/audit"
	"skill_registry/app/catalogapi/common"
	sessionhandlers "skill_registry/app/catalogapi/handlers/session"
	catalogmiddleware "skill_registry/app/catalogapi/middleware"
	routerpkg "skill_registry/app/catalogapi/router"
	"skill_registry/app/catalogapi/system"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	internalcataloghttp "skill_registry/internal/catalog/http"
	internalgovernancehttp "skill_registry/internal/governance/http"
	packagingapplication "skill_registry/internal/packaging/application"
	internalpackaginghttp "skill_registry/internal/packaging/http"
	"skill_registry/internal/platform/auth"
	"skill_registry/internal/platform/observability"
	persist "skill_registry/internal/platform/persistence"
	internalsystemhttp "skill_registry/internal/system/http"
)

type packagingApplication interface {
	packagingService
	packagingapplication.AccessService
	PublishContext(ctx context.Context, input packaging.PublishRequest) (packaging.PublishResponse, error)
	ConfigurePublishWorkflow(repo packagingapplication.ReleaseRecordCreator, publisher packagingapplication.PackagePublisher)
}

type packagingService = packagingapplication.WorkflowService

type ServerConfig struct {
	Repository                      catalog.Repository
	CatalogReadService              catalogReadService
	CatalogMutationService          catalogMutationService
	CatalogGovernanceService        catalogGovernanceService
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
	Packaging                       packagingApplication
	Idempotency                     persist.IdempotencyStore
	Audit                           persist.AuditStore
	Logger                          *log.Logger
}

type Server struct {
	repo                             catalog.Repository
	authorizer                       *auth.Authorizer
	sessionManager                   *auth.SessionManager
	accessWorkflow                   accessWorkflowHandler
	catalogReadModule                catalogReadWorkflow
	auditQueryModule                 auditQueryWorkflow
	catalogMutationService           catalogMutationService
	catalogGovernanceService         catalogGovernanceService
	listGovernanceChangesFunc        func(ctx context.Context, principal auth.Principal, query catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error)
	createReleaseRecordFunc          func(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error)
	applySkillGovernanceChangeFunc   func(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
	applyVersionGovernanceChangeFunc func(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
	packagingWorkflow                *internalpackaginghttp.Workflow
	cleanupPackagingFunc             func(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error)
	downloadBundleFunc               func(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) (string, *catalog.Error)
	sessionModule                    sessionWorkflow
	systemModule                     systemWorkflow
	publicNPMRegistryURL             string
	packaging                        packagingApplication
	auditRecorder                    *audit.Recorder
	logger                           *log.Logger
	metrics                          *observability.Collector
	requestID                        atomic.Uint64
	mode                             ServerMode
	readListCacheTTL                 time.Duration
	readDetailCacheTTL               time.Duration
	readCache                        *readCache
	readLimiter                      *requestLimiter
	writeLimiter                     *requestLimiter
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
		packagingSvc = config.Packaging
	}

	catalogReadService := config.CatalogReadService
	if catalogReadService == nil {
		catalogReadService = resolveCatalogReadService(config.Repository)
	}
	catalogMutationService := config.CatalogMutationService
	if catalogMutationService == nil {
		catalogMutationService = resolveCatalogWriteService(config.Repository)
	}
	catalogGovernanceService := config.CatalogGovernanceService
	if catalogGovernanceService == nil {
		catalogGovernanceService = resolveCatalogGovernanceService(config.Repository)
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
		service: catalogReadService,
	}
	server.catalogMutationService = catalogMutationService
	server.catalogGovernanceService = catalogGovernanceService
	server.accessWorkflow = &accessWorkflow{
		authorizer: authorizer,
		audit:      auditRecorder,
		logger:     logger,
	}
	server.auditQueryModule = audit.NewQueryModule(config.Audit, authorizer)
	server.listGovernanceChangesFunc = server.listGovernanceChanges
	server.createReleaseRecordFunc = server.createReleaseRecord
	server.applySkillGovernanceChangeFunc = server.applySkillGovernanceChange
	server.applyVersionGovernanceChangeFunc = server.applyVersionGovernanceChange
	server.packagingWorkflow = internalpackaginghttp.NewWorkflow(internalpackaginghttp.WorkflowConfig{
		Service:             packagingSvc,
		Idempotency:         config.Idempotency,
		Audit:               auditRecorder,
		CacheTTL:            10 * time.Minute,
		AllowAnonymousCache: false,
	})
	server.cleanupPackagingFunc = server.cleanupPackaging
	server.downloadBundleFunc = server.downloadBundle
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

func resolveCatalogReadService(repository catalog.Repository) catalogReadService {
	if provider, ok := repository.(catalog.ApplicationServiceProvider); ok && provider.CatalogReadService() != nil {
		return provider.CatalogReadService()
	}
	return repository
}

func resolveCatalogWriteService(repository catalog.Repository) catalogMutationService {
	if provider, ok := repository.(catalog.ApplicationServiceProvider); ok && provider.CatalogWriteService() != nil {
		return provider.CatalogWriteService()
	}
	return repository
}

func resolveCatalogGovernanceService(repository catalog.Repository) catalogGovernanceService {
	if provider, ok := repository.(catalog.ApplicationServiceProvider); ok && provider.CatalogReadService() != nil {
		return provider.CatalogReadService()
	}
	return repository
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
	systemHTTPHandlers := internalsystemhttp.New(s.systemModule.Readyz, s.metrics.RenderPrometheus)
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
	catalogHTTPHandlers := internalcataloghttp.New(internalcataloghttp.Handlers{
		AuthorizePackage:     s.authorizePackageAccess,
		ServeCachedReadJSON:  s.serveCachedReadJSON,
		ReadListCacheTTL:     s.readListCacheTTL,
		ReadDetailCacheTTL:   s.readDetailCacheTTL,
		ListAuditEvents:      s.auditQueryModule.ListEvents,
		ListSkillsFunc:       s.catalogReadModule.listSkills,
		ListCuratedSkills:    s.catalogReadModule.listCuratedSkills,
		GetSkillFunc:         s.catalogReadModule.getSkill,
		ListVersionsFunc:     s.catalogReadModule.listVersions,
		GetVersionFunc:       s.catalogReadModule.getVersion,
		ResolveInstallTarget: s.catalogReadModule.resolveInstallTarget,
		PublicInstallTarget:  s.publicInstallTarget,
	})
	governanceHTTPHandlers := internalgovernancehttp.New(internalgovernancehttp.Handlers{
		ServeCachedReadJSON: s.serveCachedReadJSON,
		ReadListCacheTTL:    s.readListCacheTTL,
		AuthorizePackage:    s.authorizePackageAccess,
		AuthorizeAction:     s.accessWorkflow.authorizeAction,
		ListGovernance:      s.listGovernanceChangesFunc,
		AcquireWriteLimiter: func() (func(), *catalog.Error) {
			return s.writeLimiter.acquire()
		},
		CreateReleaseRecord: func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
			return s.createReleaseRecordFunc(request.Context(), principal, packageName, versionPath, rawBody)
		},
		ApplySkillGovernanceChange: func(request *http.Request, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
			return s.applySkillGovernanceChangeFunc(request.Context(), principal, packageName, rawBody)
		},
		ApplyVersionGovernance: func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
			return s.applyVersionGovernanceChangeFunc(request.Context(), principal, packageName, versionPath, rawBody)
		},
	})
	packagingHTTPHandlers := internalpackaginghttp.New(internalpackaginghttp.Handlers{
		HasPackaging: func() bool {
			return s.packaging != nil
		},
		AuthorizePackage: s.authorizePackageAccess,
		AcquireWriteLimiter: func() (func(), *catalog.Error) {
			return s.writeLimiter.acquire()
		},
		DecodePrepareRequest: func(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.PrepareRequest, *catalog.Error) {
			return s.packagingWorkflow.DecodePrepareRequest(ctx, principal, rawBody)
		},
		DecodeValidateRequest: func(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.ValidateRequest, *catalog.Error) {
			return s.packagingWorkflow.DecodeValidateRequest(ctx, principal, rawBody)
		},
		DecodePackRequest: func(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.PackRequest, *catalog.Error) {
			return s.packagingWorkflow.DecodePackRequest(ctx, principal, rawBody)
		},
		Prepare: func(ctx context.Context, principal auth.Principal, payload packaging.PrepareRequest) (any, int, *catalog.Error) {
			return s.packagingWorkflow.Prepare(ctx, principal, payload)
		},
		Validate: func(ctx context.Context, principal auth.Principal, payload packaging.ValidateRequest) (any, int, *catalog.Error) {
			return s.packagingWorkflow.Validate(ctx, principal, payload)
		},
		Pack: func(ctx context.Context, principal auth.Principal, payload packaging.PackRequest) (any, int, *catalog.Error) {
			return s.packagingWorkflow.Pack(ctx, principal, payload)
		},
		Publish: func(ctx context.Context, payload packaging.PublishRequest) (packaging.PublishResponse, error) {
			if s.packaging == nil {
				return packaging.PublishResponse{}, catalog.NewError(http.StatusNotFound, "not_found", "Not found.", nil)
			}
			return s.packaging.PublishContext(ctx, payload)
		},
		Cleanup: func(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error) {
			return s.cleanupPackagingFunc(ctx, principal, payload)
		},
		DownloadBundle: func(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) (string, *catalog.Error) {
			return s.downloadBundleFunc(ctx, principal, workspacePath, packageName, version)
		},
		RecordMutationAudit: s.recordMutationAudit,
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
		AuditEvents:       s.protectedCatalogHandler(auth.ActionReadAuditEvents, catalogHTTPHandlers.AuditEvents),
		Skills:            s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.Skills),
		SkillsSearch:      s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.SkillsSearch),
		CuratedSkills:     s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.CuratedSkills),
		GovernanceChanges: s.protectedCatalogHandler(auth.ActionReadCatalog, governanceHTTPHandlers.GovernanceChanges),
		SkillDetail:       s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.SkillDetail),
		VersionList:       s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.VersionList),
		VersionDetail:     s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.VersionDetail),
		InstallTarget:     s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.InstallTarget),
		SkillGovernance:   s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.SkillGovernance),
		ReleaseRecord:     s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.ReleaseRecord),
		VersionGovernance: s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.VersionGovernance),
		PackagingPrepare:  s.protectedActionHandler(auth.ActionPackagingPrepare, packagingHTTPHandlers.PrepareHandler),
		PackagingValidate: s.protectedActionHandler(auth.ActionPackagingValidate, packagingHTTPHandlers.ValidateHandler),
		PackagingPack:     s.protectedActionHandler(auth.ActionPackagingPack, packagingHTTPHandlers.PackHandler),
		PackagingPublish:  s.protectedActionHandler(auth.ActionPackagingPublish, packagingHTTPHandlers.PublishHandler),
		PackagingCleanup:  s.protectedActionHandler(auth.ActionPackagingPack, packagingHTTPHandlers.CleanupHandler),
		PackagingBundle:   s.protectedActionHandler(auth.ActionPackagingPack, packagingHTTPHandlers.BundleHandler),
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

func (s *Server) protectedCatalogHandler(action auth.Action, handler gin.HandlerFunc) gin.HandlerFunc {
	return s.protectedActionHandler(action, handler)
}

func (s *Server) protectedActionHandler(action auth.Action, handler gin.HandlerFunc) gin.HandlerFunc {
	return catalogmiddleware.RequireAction(
		handler,
		action,
		s.accessWorkflow.authorizeAction,
		s.recordAuthenticationDeniedAudit,
	)
}

// Handler builds the HTTP routing tree with a gin engine while preserving the
// exported http.Handler contract.
func (s *Server) Handler() http.Handler {
	return routerpkg.New(s.routerConfig())
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
