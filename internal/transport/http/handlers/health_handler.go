package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"job_aggregator/internal/config"
)

type HealthHandler struct {
	config config.Config
	db     *sql.DB
}

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Database  string `json:"database"`
	Timestamp string `json:"timestamp"`
}

func NewHealthHandler(cfg config.Config, db *sql.DB) *HealthHandler {
	return &HealthHandler{
		config: cfg,
		db:     db,
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	response := healthResponse{
		Status:    "ok",
		Service:   h.config.App.Name,
		Database:  h.databaseStatus(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *HealthHandler) databaseStatus() string {
	if h.db == nil {
		return "disabled"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		return "unhealthy"
	}

	return "connected"
}
