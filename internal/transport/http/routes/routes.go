package routes

import (
	"log"
	"net/http"

	"job_aggregator/internal/transport/http/handlers"
)

func New(
	logger *log.Logger,
	healthHandler *handlers.HealthHandler,
	jobHandler *handlers.JobHandler,
	scrapeMetricHandler *handlers.ScrapeMetricHandler,
	sourceHandler *handlers.SourceHandler,
	workerHandler *handlers.WorkerHandler,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", healthHandler)
	mux.HandleFunc("GET /internal/jobs/categories", jobHandler.ListCategories)
	mux.HandleFunc("GET /internal/jobs", jobHandler.List)
	mux.HandleFunc("GET /internal/jobs/{id}", jobHandler.Get)
	mux.HandleFunc("PATCH /internal/jobs/{id}", jobHandler.Patch)
	mux.HandleFunc("POST /internal/jobs/{id}/approve", jobHandler.Approve)
	mux.HandleFunc("POST /internal/jobs/{id}/reject", jobHandler.Reject)
	mux.HandleFunc("POST /internal/jobs/{id}/archive", jobHandler.Archive)
	mux.HandleFunc("GET /internal/sources", sourceHandler.List)
	mux.HandleFunc("GET /internal/worker/scrape-health", scrapeMetricHandler.Get24hSummary)
	mux.HandleFunc("GET /internal/worker/runs", scrapeMetricHandler.ListRecentRuns)
	mux.HandleFunc("POST /internal/worker/run", workerHandler.Run)
	mux.HandleFunc("GET /internal/worker/status", workerHandler.Status)

	return loggingMiddleware(logger, mux)
}
