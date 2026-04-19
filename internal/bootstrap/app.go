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
	SourceRepository     *repository.SourceRepository
	JobRepository        *repository.JobRepository
	JobRawDataRepository *repository.JobRawDataRepository
	PipelineService      *pipeline.Service
	HealthHandler        *transporthandlers.HealthHandler
	JobHandler           *transporthandlers.JobHandler
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

	sourceRepo := repository.NewSourceRepository(db)
	jobRepo := repository.NewJobRepository(db)
	jobRawDataRepo := repository.NewJobRawDataRepository(db)
	pipelineService := pipeline.NewService(logger, sourceRepo, jobRepo, jobRawDataRepo)

	healthHandler := transporthandlers.NewHealthHandler(cfg, db)
	jobHandler := transporthandlers.NewJobHandler(logger, jobRepo)
	sourceHandler := transporthandlers.NewSourceHandler(sourceRepo)
	workerHandler := transporthandlers.NewWorkerHandler(logger, pipelineService)
	router := transportroutes.New(healthHandler, jobHandler, sourceHandler, workerHandler)

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
		SourceRepository:     sourceRepo,
		JobRepository:        jobRepo,
		JobRawDataRepository: jobRawDataRepo,
		PipelineService:      pipelineService,
		HealthHandler:        healthHandler,
		JobHandler:           jobHandler,
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
