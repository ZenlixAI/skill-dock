package catalogapi

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/audit"
	"skill_registry/app/catalogapi/common"
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
		Healthz:           s.ginHandler(http.HandlerFunc(s.handleHealthz)),
		Readyz:            s.ginHandler(http.HandlerFunc(s.handleReadyz)),
		Metrics:           s.ginHandler(http.HandlerFunc(s.handleMetrics)),
		Session:           s.ginHandler(http.HandlerFunc(s.handleSession)),
		SessionLogin:      s.ginHandler(http.HandlerFunc(s.handleSessionLogin)),
		SessionLogout:     s.ginHandler(http.HandlerFunc(s.handleSessionLogout)),
		AuditEvents:       s.ginHandler(s.requireAction(s.handleAuditEvents, auth.ActionReadAuditEvents)),
		Skills:            s.ginHandler(s.requireAction(s.handleSkills, auth.ActionReadCatalog)),
		SkillsSearch:      s.ginHandler(s.requireAction(s.handleSkills, auth.ActionReadCatalog)),
		CuratedSkills:     s.ginHandler(s.requireAction(s.handleCuratedSkills, auth.ActionReadCatalog)),
		GovernanceChanges: s.ginHandler(s.requireAction(s.handleGovernanceChanges, auth.ActionReadCatalog)),
		SkillDetail:       s.ginHandler(s.requireAction(s.handleSkillDetailRoute, auth.ActionReadCatalog), "package"),
		VersionList:       s.ginHandler(s.requireAction(s.handleVersionListRoute, auth.ActionReadCatalog), "package"),
		VersionDetail:     s.ginHandler(s.requireAction(s.handleVersionDetailRoute, auth.ActionReadCatalog), "package", "version"),
		InstallTarget:     s.ginHandler(s.requireAction(s.handleInstallTargetRoute, auth.ActionReadCatalog), "package"),
		SkillGovernance:   s.ginHandler(s.requireAction(s.handleSkillGovernanceRoute, auth.ActionReadCatalog), "package"),
		ReleaseRecord:     s.ginHandler(s.requireAction(s.handleReleaseRecordRoute, auth.ActionReadCatalog), "package", "version"),
		VersionGovernance: s.ginHandler(s.requireAction(s.handleVersionGovernanceRoute, auth.ActionReadCatalog), "package", "version"),
		PackagingPrepare:  s.ginHandler(s.requireAction(s.handlePackagingPrepare, auth.ActionPackagingPrepare)),
		PackagingValidate: s.ginHandler(s.requireAction(s.handlePackagingValidate, auth.ActionPackagingValidate)),
		PackagingPack:     s.ginHandler(s.requireAction(s.handlePackagingPack, auth.ActionPackagingPack)),
		PackagingPublish:  s.ginHandler(s.requireAction(s.handlePackagingPublish, auth.ActionPackagingPublish)),
		PackagingCleanup:  s.ginHandler(s.requireAction(s.handlePackagingCleanup, auth.ActionPackagingPack)),
		PackagingBundle:   s.ginHandler(s.requireAction(s.handlePackagingBundleDownload, auth.ActionPackagingPack)),
	}
}

func (s *Server) ginHandler(handler http.Handler, pathParams ...string) gin.HandlerFunc {
	return func(context *gin.Context) {
		request := context.Request
		writer := http.ResponseWriter(context.Writer)
		if unwrapper, ok := any(context.Writer).(interface{ Unwrap() http.ResponseWriter }); ok {
			writer = unwrapper.Unwrap()
		}
		for _, key := range pathParams {
			value := context.Param(key)
			if decoded, err := url.PathUnescape(value); err == nil {
				value = decoded
			}
			request.SetPathValue(key, value)
		}
		handler.ServeHTTP(writer, request)
		if recorder, ok := writer.(*observability.StatusRecorder); ok {
			context.Writer.WriteHeader(recorder.Status)
		}
	}
}

func (s *Server) withCanonicalRoutes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if target, ok := s.canonicalRouteTarget(request); ok {
			http.Redirect(writer, request, target, http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(writer, request)
	})
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
