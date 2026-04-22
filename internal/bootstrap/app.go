package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"job_aggregator/internal/config"
	"job_aggregator/internal/database"
	applogger "job_aggregator/internal/logger"
	"job_aggregator/internal/repository"
	"job_aggregator/internal/services/auth"
	"job_aggregator/internal/services/pipeline"
	transporthandlers "job_aggregator/internal/transport/http/handlers"
	transportroutes "job_aggregator/internal/transport/http/routes"
)

type App struct {
	Config               config.Config
	Logger               *log.Logger
	DB                   *sql.DB
	AdminUserRepository  *repository.AdminUserRepository
	AnalyticsRepository  *repository.AnalyticsRepository
	AboutPageRepository  *repository.AboutPageRepository
	SourceRepository     *repository.SourceRepository
	JobRepository        *repository.JobRepository
	JobRawDataRepository *repository.JobRawDataRepository
	ScrapeMetricRepo     *repository.ScrapeRunMetricRepository
	AuthService          *auth.Service
	PipelineService      *pipeline.Service
	HealthHandler        *transporthandlers.HealthHandler
	AnalyticsHandler     *transporthandlers.AnalyticsHandler
	AuthHandler          *transporthandlers.AuthHandler
	AboutHandler         *transporthandlers.AboutHandler
	JobHandler           *transporthandlers.JobHandler
	ScrapeMetricHandler  *transporthandlers.ScrapeMetricHandler
	SourceHandler        *transporthandlers.SourceHandler
	WorkerHandler        *transporthandlers.WorkerHandler
	Server               *http.Server
}

func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger := applogger.New()

	db, err := database.Open(context.Background(), cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	adminUserRepo := repository.NewAdminUserRepository(db)
	if err := adminUserRepo.EnsureBootstrapAdmin(context.Background(), cfg.Auth.AdminUsername, cfg.Auth.AdminPassword); err != nil {
		return nil, fmt.Errorf("ensure bootstrap admin: %w", err)
	}

	aboutPageRepo := repository.NewAboutPageRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	sourceRepo := repository.NewSourceRepository(db)
	jobRepo := repository.NewJobRepository(db)
	jobRawDataRepo := repository.NewJobRawDataRepository(db)
	scrapeMetricRepo := repository.NewScrapeRunMetricRepository(db)
	authService := auth.NewService(cfg.Auth, adminUserRepo)
	pipelineService := pipeline.NewService(logger, sourceRepo, jobRepo, jobRawDataRepo, scrapeMetricRepo)

	healthHandler := transporthandlers.NewHealthHandler(cfg, db)
	analyticsHandler := transporthandlers.NewAnalyticsHandler(logger, analyticsRepo)
	authHandler := transporthandlers.NewAuthHandler(logger, authService)
	aboutHandler := transporthandlers.NewAboutHandler(logger, aboutPageRepo)
	jobHandler := transporthandlers.NewJobHandler(logger, jobRepo)
	scrapeMetricHandler := transporthandlers.NewScrapeMetricHandler(logger, scrapeMetricRepo)
	sourceHandler := transporthandlers.NewSourceHandler(sourceRepo)
	workerHandler := transporthandlers.NewWorkerHandler(logger, pipelineService)
	router := transportroutes.New(logger, cfg.CORS, authService, healthHandler, analyticsHandler, authHandler, aboutHandler, jobHandler, scrapeMetricHandler, sourceHandler, workerHandler)

	server := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      router,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	}

	return &App{
		Config:               cfg,
		Logger:               logger,
		DB:                   db,
		AdminUserRepository:  adminUserRepo,
		AnalyticsRepository:  analyticsRepo,
		AboutPageRepository:  aboutPageRepo,
		SourceRepository:     sourceRepo,
		JobRepository:        jobRepo,
		JobRawDataRepository: jobRawDataRepo,
		ScrapeMetricRepo:     scrapeMetricRepo,
		AuthService:          authService,
		PipelineService:      pipelineService,
		HealthHandler:        healthHandler,
		AnalyticsHandler:     analyticsHandler,
		AuthHandler:          authHandler,
		AboutHandler:         aboutHandler,
		JobHandler:           jobHandler,
		ScrapeMetricHandler:  scrapeMetricHandler,
		SourceHandler:        sourceHandler,
		WorkerHandler:        workerHandler,
		Server:               server,
	}, nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := a.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := database.Close(a.DB); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	return nil
}

func (a *App) Close() error {
	return database.Close(a.DB)
}
