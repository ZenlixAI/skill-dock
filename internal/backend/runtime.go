package backend

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/app/catalogapi"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	artifactadapter "skill_registry/foundation/artifact"
	"skill_registry/foundation/persist"
	catalogapp "skill_registry/internal/catalog/application"
)

type DBPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type Config struct {
	Addr                string
	DatabaseURL         string
	SharedWorkspaceRoot string
	ArtifactSource      catalog.ArtifactSource
	RepositoryOptions   catalog.SQLRepositoryOptions
	Server              catalogapi.ServerConfig
	HTTPServer          catalogapi.HTTPServerConfig
	DBPool              DBPoolConfig
	IdempotencyTTL      time.Duration
}

type Runtime struct {
	DB           *sql.DB
	Repository   catalog.Repository
	StoreSet     *persist.SQLStoreSet
	CatalogRead  *catalogapp.ReadService
	CatalogWrite *catalogapp.WriteService
	Server       *catalogapi.Server
	Handler      http.Handler
	HTTPServer   *http.Server
}

func NewRuntime(ctx context.Context, config Config) (*Runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	databaseURL := strings.TrimSpace(config.DatabaseURL)
	if databaseURL == "" {
		databaseURL = "file:" + filepath.Join("data", "skill-registry.db")
	}
	driverName := sqlDriverNameForURL(databaseURL)

	db, err := sql.Open(driverName, databaseURL)
	if err != nil {
		return nil, err
	}
	configureDBPool(db, config.DBPool)

	repositoryOptions := config.RepositoryOptions
	if repositoryOptions.ArtifactSource == nil {
		repositoryOptions.ArtifactSource = config.ArtifactSource
	}
	repository, err := catalog.NewSQLRepositoryWithDB(ctx, db, driverName, catalog.SampleCatalogData{}, repositoryOptions)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	runtimeRepository := catalog.Repository(repository)

	idempotencyTTL := config.IdempotencyTTL
	if idempotencyTTL <= 0 {
		idempotencyTTL = 10 * time.Minute
	}
	storeSet, err := persist.NewSQLStoreSetWithDB(ctx, db, driverName, idempotencyTTL)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	packagingService := packaging.NewServiceWithWorkspaceStore(config.SharedWorkspaceRoot, storeSet.Workspace)
	packagingService.ConfigurePublishWorkflow(
		runtimeRepository,
		artifactadapter.NewNPMPublisher(
			strings.TrimRight(strings.TrimSpace(config.Server.NPMRegistryURL), "/"),
			strings.TrimSpace(config.Server.NPMRegistryPublishAuthorization),
			config.Server.NPMRegistryPublishClient,
		),
	)

	serverConfig := config.Server
	serverConfig.Repository = runtimeRepository
	serverConfig.Packaging = packagingService
	serverConfig.Idempotency = storeSet.Idempotency
	serverConfig.Audit = storeSet.Audit
	serverConfig.PackagingPublishConfigured = true

	var catalogRead *catalogapp.ReadService
	var catalogWrite *catalogapp.WriteService
	if provider, ok := any(runtimeRepository).(catalog.ApplicationServiceProvider); ok {
		catalogRead = provider.CatalogReadService()
		catalogWrite = provider.CatalogWriteService()
		serverConfig.CatalogReadService = catalogRead
		serverConfig.CatalogMutationService = catalogWrite
		if catalogRead != nil {
			serverConfig.CatalogGovernanceService = catalogRead
		}
	}

	server := catalogapi.NewServer(serverConfig)
	handler := server.Handler()
	httpServer := catalogapi.NewHTTPServer(config.Addr, handler, config.HTTPServer)

	return &Runtime{
		DB:           db,
		Repository:   runtimeRepository,
		StoreSet:     storeSet,
		CatalogRead:  catalogRead,
		CatalogWrite: catalogWrite,
		Server:       server,
		Handler:      handler,
		HTTPServer:   httpServer,
	}, nil
}

func configureDBPool(db *sql.DB, config DBPoolConfig) {
	if db == nil {
		return
	}
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

func sqlDriverNameForURL(databaseURL string) string {
	switch {
	case strings.HasPrefix(databaseURL, "postgres://"), strings.HasPrefix(databaseURL, "postgresql://"):
		return "pgx"
	default:
		return "sqlite"
	}
}
