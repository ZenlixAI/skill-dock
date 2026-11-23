package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/internal/access"
	catalog "skill_registry/internal/catalog/domain"
	internalcataloghttp "skill_registry/internal/catalog/http"
	"skill_registry/internal/favorites"
	internalgovernancehttp "skill_registry/internal/governance/http"
	"skill_registry/internal/identity"
	packaging "skill_registry/internal/packaging/domain"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	platformaccess "skill_registry/internal/platform/http/access"
	platformmiddleware "skill_registry/internal/platform/http/middleware"
	"skill_registry/internal/platform/observability"
	persist "skill_registry/internal/platform/persistence"
	"skill_registry/internal/scim"
	internalsystemhttp "skill_registry/internal/system/http"
	"skill_registry/internal/tags"
)

type ImportReviewFunc func(context.Context, packaging.ImportReviewRequest) (packaging.ImportReviewResponse, error)

type PublishFunc func(context.Context, packaging.PublishRequest) (packaging.PublishResponse, error)

type ServerConfig struct {
	Repository                          catalog.Repository
	Mode                                ServerMode
	InstallTargetMode                   InstallTargetMode
	ReadListCacheTTL                    time.Duration
	ReadDetailCacheTTL                  time.Duration
	ReadMaxInFlight                     int
	WriteMaxInFlight                    int
	ReaderToken                         string
	AuthoringToken                      string
	PublisherToken                      string
	NPMRegistryURL                      string
	NPMRegistryReadAuthorization        string
	NPMRegistryPublishAuthorization     string
	PrivateNPMRegistryURL               string
	PrivateNPMRegistryReadAuthorization string
	NPMRegistryPublishClient            *http.Client
	PublicNPMRegistryURL                string
	AuthTokens                          map[string]auth.TokenConfig
	ImportReview                        ImportReviewFunc
	Publish                             PublishFunc
	ExternalSkillSearcher               catalog.ExternalSkillSearcher
	Identity                            *identity.Service
	IdentityStore                       identity.Store
	AccessStore                         access.Store
	Access                              *access.Decider
	TagStore                            *tags.Store
	FavoriteStore                       *favorites.Store
	SCIMStore                           *scim.Store
	SCIMToken                           string
	CORSOrigins                         []string
	Idempotency                         persist.IdempotencyStore
	Audit                               persist.AuditStore
	Logger                              *log.Logger
}

type Server struct {
	authorizer                          *auth.Authorizer
	accessWorkflow                      platformaccess.Gate
	listSkillsFunc                      func(context.Context, auth.Principal, internalcataloghttp.SkillsQuery) (catalog.SkillsPage, *catalog.Error)
	listCuratedSkillsFunc               func(context.Context, auth.Principal, int, int) (catalog.CuratedSkillsPage, *catalog.Error)
	getSkillFunc                        func(context.Context, string) (catalog.SkillDetail, error)
	listVersionsFunc                    func(context.Context, catalog.VersionQuery) (catalog.VersionsPage, error)
	getVersionFunc                      func(context.Context, string, string) (catalog.SkillVersionDetail, error)
	resolveInstallTargetFunc            func(context.Context, string, string) (catalog.InstallResolution, error)
	auditListEvents                     func(context.Context, auth.Principal, url.Values) (persist.AuditPage, *catalog.Error)
	systemReadyz                        func(context.Context) (internalsystemhttp.Readyz, int)
	governanceWorkflow                  internalgovernancehttp.Workflow
	npmRegistryURL                      string
	npmRegistryReadAuthorization        string
	npmRegistryPublishAuthorization     string
	privateNPMRegistryURL               string
	privateNPMRegistryReadAuthorization string
	publicNPMRegistryURL                string
	importReview                        ImportReviewFunc
	publish                             PublishFunc
	idempotency                         persist.IdempotencyStore
	auditRecorder                       *platformaudit.Recorder
	identity                            *identity.Service
	identityStore                       identity.Store
	accessStore                         access.Store
	access                              *access.Decider
	tagStore                            *tags.Store
	favoriteStore                       *favorites.Store
	scimStore                           *scim.Store
	scimToken                           string
	corsOrigins                         []string
	logger                              *log.Logger
	metrics                             *observability.Collector
	requestID                           atomic.Uint64
	mode                                ServerMode
	installTargetMode                   InstallTargetMode
	readListCacheTTL                    time.Duration
	readDetailCacheTTL                  time.Duration
	readCache                           *platformhttp.ReadCache
	readLimiter                         *platformhttp.RequestLimiter
	writeLimiter                        *platformhttp.RequestLimiter
	externalSkillSearcher               catalog.ExternalSkillSearcher
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
	auditRecorder := platformaudit.NewRecorder(config.Audit)
	idempotencyStore := config.Idempotency
	if idempotencyStore == nil {
		idempotencyStore = newMemoryIdempotencyStore()
	}

	readService := catalog.ReadRepository(config.Repository)
	if config.ExternalSkillSearcher != nil {
		readService = newExternalReadService(readService, config.ExternalSkillSearcher)
	}

	server := &Server{
		authorizer:                          authorizer,
		npmRegistryURL:                      strings.TrimSpace(config.NPMRegistryURL),
		npmRegistryReadAuthorization:        strings.TrimSpace(config.NPMRegistryReadAuthorization),
		npmRegistryPublishAuthorization:     strings.TrimSpace(config.NPMRegistryPublishAuthorization),
		privateNPMRegistryURL:               strings.TrimSpace(config.PrivateNPMRegistryURL),
		privateNPMRegistryReadAuthorization: strings.TrimSpace(config.PrivateNPMRegistryReadAuthorization),
		publicNPMRegistryURL:                strings.TrimRight(strings.TrimSpace(config.PublicNPMRegistryURL), "/"),
		importReview:                        config.ImportReview,
		publish:                             config.Publish,
		idempotency:                         idempotencyStore,
		auditRecorder:                       auditRecorder,
		identity:                            config.Identity,
		identityStore:                       config.IdentityStore,
		accessStore:                         config.AccessStore,
		access:                              config.Access,
		tagStore:                            config.TagStore,
		favoriteStore:                       config.FavoriteStore,
		scimStore:                           config.SCIMStore,
		scimToken:                           strings.TrimSpace(config.SCIMToken),
		corsOrigins:                         config.CORSOrigins,
		logger:                              logger,
		metrics:                             observability.NewCollector(),
		mode:                                mode,
		installTargetMode:                   normalizeInstallTargetMode(config.InstallTargetMode),
		readListCacheTTL:                    readListCacheTTL,
		readDetailCacheTTL:                  readDetailCacheTTL,
		readCache:                           platformhttp.NewReadCache(),
		readLimiter:                         platformhttp.NewRequestLimiter(config.ReadMaxInFlight),
		writeLimiter:                        platformhttp.NewRequestLimiter(config.WriteMaxInFlight),
		externalSkillSearcher:               config.ExternalSkillSearcher,
	}
	readModule := internalcataloghttp.NewReadWorkflow(readService)
	server.listSkillsFunc = readModule.ListSkills
	server.listCuratedSkillsFunc = readModule.ListCuratedSkills
	server.getSkillFunc = readModule.GetSkill
	server.listVersionsFunc = readModule.ListVersions
	server.getVersionFunc = readModule.GetVersion
	server.resolveInstallTargetFunc = readModule.ResolveInstallTarget
	server.accessWorkflow = platformaccess.New(authorizer, auditRecorder, logger)
	auditQueryModule := platformaudit.NewQueryModule(config.Audit, authorizer)
	server.auditListEvents = auditQueryModule.ListEvents
	server.governanceWorkflow = internalgovernancehttp.NewWorkflow(authorizer, config.Repository, config.Repository, auditRecorder)
	server.systemReadyz = internalsystemhttp.NewModule(
		internalsystemhttp.NamedDependency{Name: "repository", Pinger: internalsystemhttp.DependencyPinger(config.Repository)},
		internalsystemhttp.NamedDependency{Name: "audit", Pinger: internalsystemhttp.DependencyPinger(config.Audit)},
		internalsystemhttp.NamedDependency{Name: "idempotency", Pinger: internalsystemhttp.DependencyPinger(idempotencyStore)},
	).Readyz
	return server
}

func (s *Server) serveCachedReadJSON(writer http.ResponseWriter, request *http.Request, ttl time.Duration, load func() (any, *catalog.Error)) {
	release, limitErr := s.readLimiter.Acquire()
	if limitErr != nil {
		platformhttp.WriteCatalogError(writer, request, limitErr)
		return
	}
	defer release()

	principal := auth.PrincipalFromContext(request.Context())
	if principal.IsUser() {
		writer.Header().Set("Cache-Control", "private, no-store")
	}
	cacheKey := s.readCacheKey(request, principal)
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
		principal.Kind,
		principal.Role,
		principal.UserID,
		principal.Subject,
		principal.TenantID,
		principal.TokenID,
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
	systemHTTPHandlers := internalsystemhttp.New(s.systemReadyz, s.metrics.RenderPrometheus)
	reqInstallTargetMode := s.installTargetMode
	if reqInstallTargetMode != InstallTargetModeDirect && reqInstallTargetMode != InstallTargetModeProxy {
		reqInstallTargetMode = InstallTargetModeProxy
	}
	catalogHTTPHandlers := internalcataloghttp.Handlers{
		AuthorizePackage:    s.accessWorkflow.AuthorizePackage,
		ServeCachedReadJSON: s.serveCachedReadJSON,
		ReadListCacheTTL:    s.readListCacheTTL,
		ReadDetailCacheTTL:  s.readDetailCacheTTL,
		ListAuditEvents:     s.auditListEvents,
		ListSkillsFunc: func(ctx context.Context, principal auth.Principal, query internalcataloghttp.SkillsQuery) (catalog.SkillsPage, *catalog.Error) {
			page, err := s.listSkillsFunc(ctx, principal, query)
			if err == nil {
				s.enrichSummaryTags(ctx, principal, page.Items)
			}
			return page, err
		},
		ListCuratedSkills: s.listCuratedSkillsFunc,
		GetSkillFunc: func(ctx context.Context, packageName string) (catalog.SkillDetail, error) {
			if err := s.checkSkillAccess(ctx, packageName, access.ActionRead); err != nil {
				return catalog.SkillDetail{}, err
			}
			detail, err := s.getSkillFunc(ctx, packageName)
			if err == nil {
				detail.Tags = s.tagsForPrincipal(ctx, packageName, auth.PrincipalFromContext(ctx))
			}
			return detail, err
		},
		ListVersionsFunc: func(ctx context.Context, query catalog.VersionQuery) (catalog.VersionsPage, error) {
			if err := s.checkSkillAccess(ctx, query.PackageName, access.ActionRead); err != nil {
				return catalog.VersionsPage{}, err
			}
			return s.listVersionsFunc(ctx, query)
		},
		GetVersionFunc: func(ctx context.Context, packageName string, version string) (catalog.SkillVersionDetail, error) {
			if err := s.checkSkillAccess(ctx, packageName, access.ActionRead); err != nil {
				return catalog.SkillVersionDetail{}, err
			}
			return s.getVersionFunc(ctx, packageName, version)
		},
		ResolveInstallTarget: func(ctx context.Context, packageName string, version string) (catalog.InstallResolution, error) {
			if err := s.checkSkillAccess(ctx, packageName, access.ActionInstall); err != nil {
				return catalog.InstallResolution{}, err
			}
			return s.resolveInstallTargetFunc(ctx, packageName, version)
		},
		PublicInstallTarget: func(request *http.Request, response catalog.InstallResolution) catalog.InstallResolution {
			private := s.isPrivateSkill(request.Context(), response.PackageName)
			if private {
				response = internalcataloghttp.RewriteInstallTargetToProxy(request, response)
				response.RegistryAccessMode = string(InstallTargetModeProxy)
				response.ArtifactAccess = "catalog_proxy"
				response.AuthorizationRequired = true
				return response
			}
			if reqInstallTargetMode == InstallTargetModeDirect {
				response.RegistryAccessMode = string(InstallTargetModeDirect)
				response.ArtifactAccess = "npm"
				return response
			}
			response = internalcataloghttp.RewriteInstallTargetToProxy(request, response)
			response.RegistryAccessMode = string(InstallTargetModeProxy)
			response.ArtifactAccess = "public_proxy"
			return response
		},
	}
	governanceHTTPHandlers := internalgovernancehttp.Handlers{
		ServeCachedReadJSON: s.serveCachedReadJSON,
		ReadListCacheTTL:    s.readListCacheTTL,
		AuthorizePackage:    s.accessWorkflow.AuthorizePackage,
		AuthorizeAction:     s.accessWorkflow.AuthorizeAction,
		ListGovernance:      s.governanceWorkflow.ListGovernanceChanges,
		AcquireWriteLimiter: func() (func(), *catalog.Error) {
			return s.writeLimiter.Acquire()
		},
		CreateReleaseRecord: func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
			cleanup, accessErr := s.ensurePublishAccess(request.Context(), principal, packageName, "")
			if accessErr != nil {
				return catalog.ReleaseRecordResult{}, 0, platformhttp.CatalogErrorFromError(accessErr)
			}
			result, status, err := s.governanceWorkflow.CreateReleaseRecord(request.Context(), principal, packageName, versionPath, rawBody)
			if err != nil {
				cleanup()
			}
			s.readCache.InvalidateAll()
			return result, status, err
		},
		ApplySkillGovernanceChange: func(request *http.Request, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
			return s.governanceWorkflow.ApplySkillGovernanceChange(request.Context(), principal, packageName, rawBody)
		},
		ApplyVersionGovernance: func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
			return s.governanceWorkflow.ApplyVersionGovernanceChange(request.Context(), principal, packageName, versionPath, rawBody)
		},
	}
	engine := platformhttp.NewEngine([]gin.HandlerFunc{
		platformmiddleware.WithRequestMetadata(&s.requestID),
		platformmiddleware.WithObservability(s.metrics, s.logger),
		platformmiddleware.WithAuthentication(s.authenticateRequest),
	})
	if len(s.corsOrigins) > 0 {
		engine.Use(s.corsMiddleware())
	}

	s.registerIdentityRoutes(engine)
	s.registerOwnershipRoutes(engine)
	s.registerTagRoutes(engine)
	s.registerFavoriteRoutes(engine)
	s.registerSCIMRoutes(engine)

	internalsystemhttp.RegisterRoutes(engine, internalsystemhttp.Config{
		Healthz: systemHTTPHandlers.Healthz,
		Readyz:  systemHTTPHandlers.Readyz,
		Metrics: systemHTTPHandlers.Metrics,
	})

	proxy := newNPMRegistryProxy(s.npmRegistryURL, s.publicNPMRegistryURL, s.npmRegistryReadAuthorization, s.npmRegistryPublishAuthorization)
	privateProxy := http.Handler(nil)
	if s.privateNPMRegistryURL != "" {
		privateProxy = newNPMRegistryProxy(s.privateNPMRegistryURL, s.publicNPMRegistryURL, s.privateNPMRegistryReadAuthorization, s.npmRegistryPublishAuthorization)
	}
	proxyHandler := s.authenticatedArtifactProxy(proxy, privateProxy)
	engine.Any(npmRegistryMountPath, proxyHandler)
	engine.Any(npmRegistryMountPath+"/*path", proxyHandler)

	internalcataloghttp.RegisterRoutes(engine, internalcataloghttp.Config{
		ServesReadRoutes:  s.mode.servesReadRoutes(),
		ServesWriteRoutes: s.mode.servesWriteRoutes(),
		AuditEvents:       s.protectedActionHandler(auth.ActionReadAuditEvents, catalogHTTPHandlers.AuditEvents),
		Skills:            s.protectedActionHandler(auth.ActionReadCatalog, catalogHTTPHandlers.Skills),
		SkillsSearch:      s.protectedActionHandler(auth.ActionReadCatalog, catalogHTTPHandlers.SkillsSearch),
		CuratedSkills:     s.protectedActionHandler(auth.ActionReadCatalog, catalogHTTPHandlers.CuratedSkills),
		SkillDetail:       s.protectedActionHandler(auth.ActionReadCatalog, catalogHTTPHandlers.SkillDetail),
		VersionList:       s.protectedActionHandler(auth.ActionReadCatalog, catalogHTTPHandlers.VersionList),
		VersionDetail:     s.protectedActionHandler(auth.ActionReadCatalog, catalogHTTPHandlers.VersionDetail),
		InstallTarget:     s.protectedActionHandler(auth.ActionReadCatalog, catalogHTTPHandlers.InstallTarget),
	})
	s.registerPublicRegistryRoutes(engine)

	internalgovernancehttp.RegisterRoutes(engine, internalgovernancehttp.Config{
		ServesReadRoutes:  s.mode.servesReadRoutes(),
		ServesWriteRoutes: s.mode.servesWriteRoutes(),
		GovernanceChanges: s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.GovernanceChanges),
		SkillGovernance:   s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.SkillGovernance),
		ReleaseRecord:     s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.ReleaseRecord),
		VersionGovernance: s.protectedActionHandler(auth.ActionReadCatalog, governanceHTTPHandlers.VersionGovernance),
	})

	if s.mode.servesWriteRoutes() {
		engine.POST("/api/v1/skill-packaging/import-review", s.protectedActionHandler(auth.ActionPackagingPack, s.importReviewHandler()))
		engine.POST("/api/v1/skill-packaging/publish", s.protectedActionHandler(auth.ActionPackagingPublish, func(c *gin.Context) {
			if s.publish == nil {
				c.Status(http.StatusNotFound)
				return
			}

			var payload packaging.PublishRequest
			if err := platformhttp.DecodeJSONBody(c.Request, &payload); err != nil {
				s.recordMutationAudit(c.Request, "packaging.publish", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
				platformhttp.WriteCatalogError(c.Writer, c.Request, err)
				return
			}
			if !s.accessWorkflow.AuthorizePackage(c.Writer, c.Request, payload.Target.PackageName) {
				return
			}
			cleanup, accessErr := s.ensurePublishAccess(c.Request.Context(), auth.PrincipalFromContext(c.Request.Context()), payload.Target.PackageName, payload.Options.Visibility)
			if accessErr != nil {
				catalogErr := platformhttp.CatalogErrorFromError(accessErr)
				s.recordMutationAudit(c.Request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
				platformhttp.WriteCatalogError(c.Writer, c.Request, catalogErr)
				return
			}
			release, err := s.writeLimiter.Acquire()
			if err != nil {
				cleanup()
				platformhttp.WriteCatalogError(c.Writer, c.Request, err)
				return
			}
			defer release()

			result, publishErr := s.publish(c.Request.Context(), payload)
			s.readCache.InvalidateAll()
			if publishErr != nil {
				cleanup()
				catalogErr := platformhttp.CatalogErrorFromError(publishErr)
				s.recordMutationAudit(c.Request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
				platformhttp.WriteCatalogError(c.Writer, c.Request, catalogErr)
				return
			}

			status := http.StatusOK
			if result.Created {
				status = http.StatusCreated
			}
			s.recordMutationAudit(c.Request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, status, "succeeded", map[string]any{
				"publishedToRegistry": result.PublishedToRegistry,
				"created":             result.Created,
				"idempotentReplay":    result.IdempotentReplay,
			})
			platformhttp.WriteJSON(c.Writer, status, result)
		}))
	}

	return engine
}

func (s *Server) importReviewHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.importReview == nil {
			c.Status(http.StatusNotFound)
			return
		}

		var (
			principal auth.Principal
			payload   packaging.ImportReviewRequest
		)

		platformhttp.ExecuteRawBodyMutation(c.Writer, c.Request, platformhttp.RawBodyMutationConfig{
			Prepare: func(ctx context.Context, rawBody []byte) *catalog.Error {
				principal = auth.PrincipalFromContext(ctx)
				if err := platformhttp.DecodeJSONBytes(rawBody, &payload); err != nil {
					s.recordMutationAudit(c.Request, "packaging.import-review", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
					return err
				}
				return nil
			},
			Authorize: func([]byte) bool {
				if packageName := payload.Target.PackageName; packageName != "" {
					return s.accessWorkflow.AuthorizePackage(c.Writer, c.Request, packageName)
				}
				return true
			},
			AcquireWriteLimiter: func() (func(), *catalog.Error) {
				return s.writeLimiter.Acquire()
			},
			Execute: func(ctx context.Context, _ []byte) (any, int, *catalog.Error) {
				requestHash, err := platformhttp.HashRequestPayload(payload)
				if err != nil {
					return nil, 0, catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil)
				}

				idempotencyKey := strings.TrimSpace(payload.GetIdempotencyKey())
				if idempotencyKey != "" && s.idempotency != nil {
					record, lookupErr := s.idempotency.Lookup(ctx, "import-review", principal.Role, idempotencyKey)
					switch {
					case lookupErr == nil:
						if record.RequestHash != requestHash {
							cacheErr := catalog.NewError(http.StatusConflict, "idempotency_key_conflict", "idempotencyKey cannot be reused with a different request body.", map[string]any{"idempotencyKey": idempotencyKey})
							s.recordMutationAudit(c.Request, "packaging.import-review", payload.Target.PackageName, payload.Target.Version, cacheErr.Status, "failed", map[string]any{"errorCode": cacheErr.Code})
							return nil, cacheErr.Status, cacheErr
						}
						s.recordMutationAudit(c.Request, "packaging.import-review", payload.Target.PackageName, payload.Target.Version, record.Status, "succeeded", map[string]any{"idempotentReplay": true})
						return json.RawMessage(append([]byte(nil), record.ResponseBody...)), record.Status, nil
					case !errors.Is(lookupErr, persist.ErrIdempotencyRecordNotFound):
						catalogErr := catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil)
						s.recordMutationAudit(c.Request, "packaging.import-review", payload.Target.PackageName, payload.Target.Version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
						return nil, catalogErr.Status, catalogErr
					}
				}

				response, reviewErr := s.importReview(ctx, payload)
				if reviewErr != nil {
					catalogErr := platformhttp.CatalogErrorFromError(reviewErr)
					s.recordMutationAudit(c.Request, "packaging.import-review", payload.Target.PackageName, payload.Target.Version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
					return nil, platformhttp.StatusFromError(reviewErr), catalogErr
				}

				if idempotencyKey == "" || s.idempotency == nil {
					s.recordMutationAudit(c.Request, "packaging.import-review", payload.Target.PackageName, payload.Target.Version, http.StatusOK, "succeeded", nil)
					return response, http.StatusOK, nil
				}

				responseBody, marshalErr := json.Marshal(response)
				if marshalErr != nil {
					return nil, 0, catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil)
				}

				record := persist.IdempotencyRecord{
					RequestHash:  requestHash,
					Status:       http.StatusOK,
					ResponseBody: append([]byte(nil), responseBody...),
					CreatedAt:    time.Now().UTC(),
				}
				_ = s.idempotency.DeleteExpired(ctx)
				saved, saveErr := s.idempotency.Save(ctx, "import-review", principal.Role, idempotencyKey, record)
				if saveErr == nil && saved {
					responseBody = append([]byte(nil), record.ResponseBody...)
				} else if saveErr == nil {
					if existing, lookupErr := s.idempotency.Lookup(ctx, "import-review", principal.Role, idempotencyKey); lookupErr == nil && existing.RequestHash == requestHash {
						responseBody = append([]byte(nil), existing.ResponseBody...)
						record.Status = existing.Status
					}
				}

				s.recordMutationAudit(c.Request, "packaging.import-review", payload.Target.PackageName, payload.Target.Version, record.Status, "succeeded", map[string]any{"idempotencyKey": idempotencyKey})
				return json.RawMessage(responseBody), record.Status, nil
			},
		})
	}
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
		if current.Kind == "" {
			current.Kind = auth.PrincipalKindService
		}
		return current, true
	}
	if s.identity != nil {
		token, ok := bearerToken(request)
		if ok {
			current, recognized, err := s.identity.AuthenticateBearer(request.Context(), token)
			if recognized && err == nil {
				return current, true
			}
		}
	}
	return auth.Principal{}, false
}

func bearerToken(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	if !strings.HasPrefix(authorization, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	return token, token != ""
}
