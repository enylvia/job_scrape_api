package routes

import (
	"net/http"

	"job_aggregator/internal/transport/http/handlers"
)

func New(healthHandler *handlers.HealthHandler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", healthHandler)

	return mux
}
