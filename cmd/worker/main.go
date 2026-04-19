package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"job_aggregator/internal/bootstrap"
	"job_aggregator/internal/services/collector"
	"job_aggregator/internal/services/collector/browsercollector"
	"job_aggregator/internal/services/collector/httpcollector"
	"job_aggregator/internal/services/collector/sources"
	"job_aggregator/internal/services/deduplicator"
	"job_aggregator/internal/services/normalizer"
)

func main() {
	app, err := bootstrap.NewApp()
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			app.Logger.Printf("close app: %v", err)
		}
	}()

	service := collector.NewService(
		app.Logger,
		app.SourceRepository,
		app.JobRepository,
		app.JobRawDataRepository,
		[]collector.Collector{
			httpcollector.New(),
			browsercollector.New(),
		},
		[]collector.SourceScraper{
			sources.NewDeallsScraper(),
			sources.NewGlintsScraper(),
		},
	)
	normalizerService := normalizer.NewService(app.Logger, app.JobRepository)
	deduplicatorService := deduplicator.NewService(app.Logger, app.JobRepository)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := service.RunOnce(ctx); err != nil {
		app.Logger.Fatalf("run collector worker: %v", err)
	}

	if err := normalizerService.RunOnce(ctx); err != nil {
		app.Logger.Fatalf("run normalizer worker: %v", err)
	}

	if err := deduplicatorService.RunOnce(ctx); err != nil {
		app.Logger.Fatalf("run deduplicator worker: %v", err)
	}
}
