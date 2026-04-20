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
	"job_aggregator/internal/services/pipeline"
	transporthandlers "job_aggregator/internal/transport/http/handlers"
	transportroutes "job_aggregator/internal/transport/http/routes"
)

type App struct {
	Config               config.Config
	Logger               *log.Logger
	DB                   *sql.DB
	AboutPageRepository  *repository.AboutPageRepository
	SourceRepository     *repository.SourceRepository
	JobRepository        *repository.JobRepository
	JobRawDataRepository *repository.JobRawDataRepository
	ScrapeMetricRepo     *repository.ScrapeRunMetricRepository
	PipelineService      *pipeline.Service
	HealthHandler        *transporthandlers.HealthHandler
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

	aboutPageRepo := repository.NewAboutPageRepository(db)
	sourceRepo := repository.NewSourceRepository(db)
	jobRepo := repository.NewJobRepository(db)
	jobRawDataRepo := repository.NewJobRawDataRepository(db)
	scrapeMetricRepo := repository.NewScrapeRunMetricRepository(db)
	pipelineService := pipeline.NewService(logger, sourceRepo, jobRepo, jobRawDataRepo, scrapeMetricRepo)

	healthHandler := transporthandlers.NewHealthHandler(cfg, db)
	aboutHandler := transporthandlers.NewAboutHandler(logger, aboutPageRepo)
	jobHandler := transporthandlers.NewJobHandler(logger, jobRepo)
	scrapeMetricHandler := transporthandlers.NewScrapeMetricHandler(logger, scrapeMetricRepo)
	sourceHandler := transporthandlers.NewSourceHandler(sourceRepo)
	workerHandler := transporthandlers.NewWorkerHandler(logger, pipelineService)
	router := transportroutes.New(logger, healthHandler, aboutHandler, jobHandler, scrapeMetricHandler, sourceHandler, workerHandler)

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
		AboutPageRepository:  aboutPageRepo,
		SourceRepository:     sourceRepo,
		JobRepository:        jobRepo,
		JobRawDataRepository: jobRawDataRepo,
		ScrapeMetricRepo:     scrapeMetricRepo,
		PipelineService:      pipelineService,
		HealthHandler:        healthHandler,
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
