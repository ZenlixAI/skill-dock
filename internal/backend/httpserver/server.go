package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	catalog "skill_registry/internal/catalog/domain"
	internalcataloghttp "skill_registry/internal/catalog/http"
	internalgovernancehttp "skill_registry/internal/governance/http"
	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	internalpackaginghttp "skill_registry/internal/packaging/http"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	platformaccess "skill_registry/internal/platform/http/access"
	platformmiddleware "skill_registry/internal/platform/http/middleware"
	platformsessionhttp "skill_registry/internal/platform/http/session"
	"skill_registry/internal/platform/observability"
	persist "skill_registry/internal/platform/persistence"
	platformsession "skill_registry/internal/platform/session"
	internalsystemhttp "skill_registry/internal/system/http"
)

type packagingApplication interface {
	packagingapplication.ReviewService
	PublishContext(ctx context.Context, input packaging.PublishRequest) (packaging.PublishResponse, error)
	ConfigurePublishWorkflow(repo packagingapplication.ReleaseRecordCreator, publisher packagingapplication.PackagePublisher)
}
type catalogReadService = internalcataloghttp.ReadService
type catalogMutationService = internalgovernancehttp.MutationService
type catalogGovernanceService = internalgovernancehttp.QueryService

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
	NPMRegistryReadAuthorization    string
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
	repo                            catalog.Repository
	authorizer                      *auth.Authorizer
	sessionManager                  *auth.SessionManager
	accessWorkflow                  platformaccess.Gate
	catalogReadModule               internalcataloghttp.Workflow
	auditQueryModule                auditQueryWorkflow
	catalogMutationService          catalogMutationService
	catalogGovernanceService        catalogGovernanceService
	governanceWorkflow              internalgovernancehttp.Workflow
	oneShotPackagingWorkflow        *internalpackaginghttp.OneShotWorkflow
	sessionModule                   platformsession.Workflow
	systemModule                    systemWorkflow
	npmRegistryURL                  string
	npmRegistryReadAuthorization    string
	npmRegistryPublishAuthorization string
	publicNPMRegistryURL            string
	packaging                       packagingApplication
	auditRecorder                   *platformaudit.Recorder
	logger                          *log.Logger
	metrics                         *observability.Collector
	requestID                       atomic.Uint64
	mode                            ServerMode
	readListCacheTTL                time.Duration
	readDetailCacheTTL              time.Duration
	readCache                       *platformhttp.ReadCache
	readLimiter                     *platformhttp.RequestLimiter
	writeLimiter                    *platformhttp.RequestLimiter
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
	auditRecorder := platformaudit.NewRecorder(config.Audit)
	var packagingReviewSvc packagingapplication.ReviewService
	if config.Packaging != nil {
		packagingReviewSvc = config.Packaging
	}

	catalogReadService := config.CatalogReadService
	if catalogReadService == nil {
		catalogReadService = config.Repository
	}
	catalogMutationService := config.CatalogMutationService
	if catalogMutationService == nil {
		catalogMutationService = config.Repository
	}
	catalogGovernanceService := config.CatalogGovernanceService
	if catalogGovernanceService == nil {
		catalogGovernanceService = config.Repository
	}

	server := &Server{
		repo:                            config.Repository,
		authorizer:                      authorizer,
		sessionManager:                  sessionManager,
		npmRegistryURL:                  strings.TrimSpace(config.NPMRegistryURL),
		npmRegistryReadAuthorization:    strings.TrimSpace(config.NPMRegistryReadAuthorization),
		npmRegistryPublishAuthorization: strings.TrimSpace(config.NPMRegistryPublishAuthorization),
		publicNPMRegistryURL:            strings.TrimRight(strings.TrimSpace(config.PublicNPMRegistryURL), "/"),
		packaging:                       config.Packaging,
		auditRecorder:                   auditRecorder,
		logger:                          logger,
		metrics:                         observability.NewCollector(),
		mode:                            mode,
		readListCacheTTL:                readListCacheTTL,
		readDetailCacheTTL:              readDetailCacheTTL,
		readCache:                       platformhttp.NewReadCache(),
		readLimiter:                     platformhttp.NewRequestLimiter(config.ReadMaxInFlight),
		writeLimiter:                    platformhttp.NewRequestLimiter(config.WriteMaxInFlight),
	}
	server.catalogReadModule = internalcataloghttp.NewReadWorkflow(catalogReadService)
	server.catalogMutationService = catalogMutationService
	server.catalogGovernanceService = catalogGovernanceService
	server.accessWorkflow = platformaccess.New(authorizer, auditRecorder, logger)
	server.auditQueryModule = platformaudit.NewQueryModule(config.Audit, authorizer)
	server.governanceWorkflow = internalgovernancehttp.NewWorkflow(authorizer, catalogMutationService, catalogGovernanceService, auditRecorder)
	server.oneShotPackagingWorkflow = internalpackaginghttp.NewOneShotWorkflow(internalpackaginghttp.OneShotWorkflowConfig{
		ReviewService:       packagingReviewSvc,
		Idempotency:         config.Idempotency,
		Audit:               auditRecorder,
		CacheTTL:            10 * time.Minute,
		AllowAnonymousCache: false,
	})
	server.sessionModule = platformsession.NewModule(authorizer, sessionManager)
	server.systemModule = internalsystemhttp.NewModule(
		internalsystemhttp.NamedDependency{Name: "repository", Pinger: internalsystemhttp.DependencyPinger(config.Repository)},
		internalsystemhttp.NamedDependency{Name: "audit", Pinger: internalsystemhttp.DependencyPinger(config.Audit)},
		internalsystemhttp.NamedDependency{Name: "idempotency", Pinger: internalsystemhttp.DependencyPinger(config.Idempotency)},
	)
	return server
}

func (s *Server) authorizePackageAccess(writer http.ResponseWriter, request *http.Request, packageName string) bool {
	return s.accessWorkflow.AuthorizePackage(writer, request, packageName)
}

func (s *Server) serveCachedReadJSON(writer http.ResponseWriter, request *http.Request, ttl time.Duration, load func() (any, *catalog.Error)) {
	release, limitErr := s.readLimiter.Acquire()
	if limitErr != nil {
		platformhttp.WriteCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	cacheKey := s.readCacheKey(request, auth.PrincipalFromContext(request.Context()))
	if cached, ok := s.readCache.Lookup(cacheKey); ok {
		platformhttp.WriteJSONBytes(writer, http.StatusOK, cached)
		return
	}

	response, err := load()
	if err != nil {
		platformhttp.WriteCatalogError(writer, request, err)
		return
	}

	raw, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		platformhttp.WriteCatalogError(writer, request, marshalErr)
		return
	}
	s.readCache.Store(cacheKey, raw, ttl)
	platformhttp.WriteJSONBytes(writer, http.StatusOK, raw)
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
		platformhttp.RequestIDFromContext(request.Context()),
		platformhttp.TraceIDFromContext(request.Context()),
		action,
		observability.AuthDecisionAllowed,
		err.Error(),
	)
}

func (s *Server) newEngine() *gin.Engine {
	systemHTTPHandlers := internalsystemhttp.New(s.systemModule.Readyz, s.metrics.RenderPrometheus)
	sessionHTTPHandlers := platformsessionhttp.Handlers{
		Authenticate:    s.authenticateRequest,
		AuthorizeAction: s.accessWorkflow.AuthorizeAction,
		CurrentFunc: func(principal auth.Principal) any {
			if s.sessionModule == nil {
				return nil
			}
			return s.sessionModule.Current(principal)
		},
		LoginFunc: func(request *http.Request, username string, password string) (auth.Principal, *http.Cookie, bool) {
			if s.sessionModule == nil {
				return auth.Principal{}, nil, false
			}
			return s.sessionModule.Login(request, username, password)
		},
		LogoutCookie: func(request *http.Request) *http.Cookie {
			if s.sessionModule == nil {
				return nil
			}
			return s.sessionModule.LogoutCookie(request)
		},
		Enabled: func() bool {
			return s.sessionModule != nil && s.sessionModule.Enabled()
		},
	}
	catalogHTTPHandlers := internalcataloghttp.Handlers{
		AuthorizePackage:     s.authorizePackageAccess,
		ServeCachedReadJSON:  s.serveCachedReadJSON,
		ReadListCacheTTL:     s.readListCacheTTL,
		ReadDetailCacheTTL:   s.readDetailCacheTTL,
		ListAuditEvents:      s.auditQueryModule.ListEvents,
		ListSkillsFunc:       s.catalogReadModule.ListSkills,
		ListCuratedSkills:    s.catalogReadModule.ListCuratedSkills,
		GetSkillFunc:         s.catalogReadModule.GetSkill,
		ListVersionsFunc:     s.catalogReadModule.ListVersions,
		GetVersionFunc:       s.catalogReadModule.GetVersion,
		ResolveInstallTarget: s.catalogReadModule.ResolveInstallTarget,
		PublicInstallTarget: func(request *http.Request, response catalog.InstallResolution) catalog.InstallResolution {
			return internalcataloghttp.RewriteInstallTargetToProxy(request, response)
		},
	}
	governanceHTTPHandlers := internalgovernancehttp.Handlers{
		ServeCachedReadJSON: s.serveCachedReadJSON,
		ReadListCacheTTL:    s.readListCacheTTL,
		AuthorizePackage:    s.authorizePackageAccess,
		AuthorizeAction:     s.accessWorkflow.AuthorizeAction,
		ListGovernance:      s.governanceWorkflow.ListGovernanceChanges,
		AcquireWriteLimiter: func() (func(), *catalog.Error) {
			return s.writeLimiter.Acquire()
		},
		CreateReleaseRecord: func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
			return s.governanceWorkflow.CreateReleaseRecord(request.Context(), principal, packageName, versionPath, rawBody)
		},
		ApplySkillGovernanceChange: func(request *http.Request, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
			return s.governanceWorkflow.ApplySkillGovernanceChange(request.Context(), principal, packageName, rawBody)
		},
		ApplyVersionGovernance: func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
			return s.governanceWorkflow.ApplyVersionGovernanceChange(request.Context(), principal, packageName, versionPath, rawBody)
		},
	}
	packagingHTTPHandlers := internalpackaginghttp.Handlers{
		HasPackaging: func() bool {
			return s.packaging != nil
		},
		AuthorizePackage: s.authorizePackageAccess,
		AcquireWriteLimiter: func() (func(), *catalog.Error) {
			return s.writeLimiter.Acquire()
		},
		DecodeImportReviewRequest: func(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.ImportReviewRequest, *catalog.Error) {
			return s.oneShotPackagingWorkflow.DecodeImportReviewRequest(ctx, principal, rawBody)
		},
		ImportReview: func(ctx context.Context, principal auth.Principal, payload packaging.ImportReviewRequest) (any, int, *catalog.Error) {
			return s.oneShotPackagingWorkflow.ImportReview(ctx, principal, payload)
		},
		Publish: func(ctx context.Context, payload packaging.PublishRequest) (packaging.PublishResponse, error) {
			if s.packaging == nil {
				return packaging.PublishResponse{}, catalog.NewError(http.StatusNotFound, "not_found", "Not found.", nil)
			}
			return s.packaging.PublishContext(ctx, payload)
		},
		RecordMutationAudit: s.recordMutationAudit,
	}

	engine := platformhttp.NewEngine([]gin.HandlerFunc{
		platformmiddleware.WithCORS(),
		platformmiddleware.WithRequestMetadata(&s.requestID),
		platformmiddleware.WithObservability(s.metrics, s.logger),
		platformmiddleware.WithCanonicalRedirect(),
		platformmiddleware.WithAuthentication(s.authenticateRequest),
	})

	internalsystemhttp.RegisterRoutes(engine, internalsystemhttp.Config{
		Healthz: systemHTTPHandlers.Healthz,
		Readyz:  systemHTTPHandlers.Readyz,
		Metrics: systemHTTPHandlers.Metrics,
	})

	proxy := newNPMRegistryProxy(s.npmRegistryURL, s.publicNPMRegistryURL, s.npmRegistryReadAuthorization, s.npmRegistryPublishAuthorization)
	engine.Any(npmRegistryMountPath, gin.WrapH(proxy))
	engine.Any(npmRegistryMountPath+"/*path", gin.WrapH(proxy))

	group := engine.Group("/api/v1/session")
	group.GET("", sessionHTTPHandlers.Current)
	group.POST("/login", sessionHTTPHandlers.Login)
	group.POST("/logout", sessionHTTPHandlers.Logout)

	internalcataloghttp.RegisterRoutes(engine, internalcataloghttp.Config{
		ServesReadRoutes:  s.mode.servesReadRoutes(),
		ServesWriteRoutes: s.mode.servesWriteRoutes(),
		AuditEvents:       s.protectedCatalogHandler(auth.ActionReadAuditEvents, catalogHTTPHandlers.AuditEvents),
		Skills:            s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.Skills),
		SkillsSearch:      s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.SkillsSearch),
		CuratedSkills:     s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.CuratedSkills),
		SkillDetail:       s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.SkillDetail),
		VersionList:       s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.VersionList),
		VersionDetail:     s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.VersionDetail),
		InstallTarget:     s.protectedCatalogHandler(auth.ActionReadCatalog, catalogHTTPHandlers.InstallTarget),
	})

	internalgovernancehttp.RegisterRoutes(engine, internalgovernancehttp.Config{
		ServesReadRoutes:  s.mode.servesReadRoutes(),
		ServesWriteRoutes: s.mode.servesWriteRoutes(),
		GovernanceChanges: s.protectedCatalogHandler(auth.ActionReadCatalog, governanceHTTPHandlers.GovernanceChanges),
		SkillGovernance:   s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.SkillGovernance),
		ReleaseRecord:     s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.ReleaseRecord),
		VersionGovernance: s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.VersionGovernance),
	})

	internalpackaginghttp.RegisterRoutes(engine, internalpackaginghttp.Config{
		ServesWriteRoutes: s.mode.servesWriteRoutes(),
		ImportReview:      s.protectedActionHandler(auth.ActionPackagingPack, packagingHTTPHandlers.ImportReviewHandler),
		Publish:           s.protectedActionHandler(auth.ActionPackagingPublish, packagingHTTPHandlers.PublishHandler),
	})

	return engine
}

func (s *Server) protectedCatalogHandler(action auth.Action, handler gin.HandlerFunc) gin.HandlerFunc {
	return s.protectedActionHandler(action, handler)
}

func (s *Server) protectedActionHandler(action auth.Action, handler gin.HandlerFunc) gin.HandlerFunc {
	return platformmiddleware.RequireAction(
		handler,
		action,
		s.accessWorkflow.AuthorizeAction,
		s.recordAuthenticationDeniedAudit,
	)
}

// Handler builds the HTTP routing tree with a gin engine while preserving the
// exported http.Handler contract.
func (s *Server) Handler() http.Handler {
	return s.newEngine()
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
