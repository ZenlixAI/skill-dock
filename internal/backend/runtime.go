package backend

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/internal/backend/httpserver"
	catalog "skill_registry/internal/catalog/domain"
	cataloginfra "skill_registry/internal/catalog/infra"
	packagingapplication "skill_registry/internal/packaging/application"
	platformartifact "skill_registry/internal/platform/artifact"
	platformpersist "skill_registry/internal/platform/persistence"
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
	RepositoryOptions   cataloginfra.SQLRepositoryOptions
	Server              httpserver.ServerConfig
	HTTPServer          httpserver.HTTPServerConfig
	DBPool              DBPoolConfig
	IdempotencyTTL      time.Duration
}

type Runtime struct {
	DB         *sql.DB
	Repository catalog.Repository
	StoreSet   *platformpersist.SQLStoreSet
	Server     *httpserver.Server
	Handler    http.Handler
	HTTPServer *http.Server
}

func NewRuntime(ctx context.Context, config Config) (*Runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	databaseURL := strings.TrimSpace(config.DatabaseURL)
	if databaseURL == "" {
		databaseURL = "file:" + filepath.Join("data", "skill-registry.db")
	}
	driverName := "sqlite"
	if strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://") {
		driverName = "pgx"
	}

	db, err := sql.Open(driverName, databaseURL)
	if err != nil {
		return nil, err
	}
	configureDBPool(db, config.DBPool)

	repositoryOptions := config.RepositoryOptions
	if repositoryOptions.ArtifactSource == nil {
		repositoryOptions.ArtifactSource = config.ArtifactSource
	}
	if repositoryOptions.ArtifactSource == nil {
		repositoryOptions.ArtifactSource = cataloginfra.NewStaticArtifactSource(nil)
	}

	internalStateStore, err := cataloginfra.PrepareSQLStateStore(ctx, db, driverName)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	repository := cataloginfra.NewRepository(cataloginfra.RepositoryConfig{
		StateStore:     internalStateStore,
		ArtifactSource: repositoryOptions.ArtifactSource,
	})
	runtimeRepository := catalog.Repository(repository)
	if !repositoryOptions.DisableSeed {
		if seedErr := repository.Seed(ctx, cataloginfra.SampleData()); seedErr != nil {
			_ = db.Close()
			return nil, seedErr
		}
	}

	idempotencyTTL := config.IdempotencyTTL
	if idempotencyTTL <= 0 {
		idempotencyTTL = 10 * time.Minute
	}
	storeSet, err := platformpersist.NewSQLStoreSetWithDB(ctx, db, driverName, idempotencyTTL)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	publisher := platformartifact.NewNPMPublisher(
		strings.TrimRight(strings.TrimSpace(config.Server.NPMRegistryURL), "/"),
		strings.TrimSpace(config.Server.NPMRegistryPublishAuthorization),
		config.Server.NPMRegistryPublishClient,
	)
	workspaceService := packagingapplication.NewWorkspaceService(
		config.SharedWorkspaceRoot,
		storeSet.Workspace,
	)
	importReviewService := packagingapplication.NewImportReviewService(workspaceService)
	publishService := packagingapplication.NewPublishService(
		workspaceService,
		runtimeRepository.CreateReleaseRecord,
		publisher.PublishPackage,
	)

	serverConfig := config.Server
	serverConfig.Repository = runtimeRepository
	serverConfig.ImportReview = importReviewService.ReviewContext
	serverConfig.Publish = publishService.PublishContext
	serverConfig.Idempotency = storeSet.Idempotency
	serverConfig.Audit = storeSet.Audit

	server := httpserver.NewServer(serverConfig)
	handler := server.Handler()
	httpServer := httpserver.NewHTTPServer(config.Addr, handler, config.HTTPServer)

	return &Runtime{
		DB:         db,
		Repository: runtimeRepository,
		StoreSet:   storeSet,
		Server:     server,
		Handler:    handler,
		HTTPServer: httpServer,
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
