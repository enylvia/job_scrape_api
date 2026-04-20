package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"job_aggregator/internal/repository"
)

type ScrapeMetricHandler struct {
	logger     *log.Logger
	metricRepo *repository.ScrapeRunMetricRepository
}

func NewScrapeMetricHandler(logger *log.Logger, metricRepo *repository.ScrapeRunMetricRepository) *ScrapeMetricHandler {
	return &ScrapeMetricHandler{
		logger:     logger,
		metricRepo: metricRepo,
	}
}

func (h *ScrapeMetricHandler) Get24hSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.metricRepo.Get24hSummary(r.Context(), time.Now().UTC())
	if err != nil {
		h.logger.Printf("scrape metric handler: get 24h summary error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to get 24h scrape summary")
		return
	}

	writeData(w, http.StatusOK, "24h scrape summary fetched successfully", 1, summary)
}

func (h *ScrapeMetricHandler) ListRecentRuns(w http.ResponseWriter, r *http.Request) {
	limit, err := parseRunsLimit(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	metrics, totalCount, err := h.metricRepo.ListRecent(r.Context(), limit)
	if err != nil {
		h.logger.Printf("scrape metric handler: list recent runs error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to list recent worker runs")
		return
	}

	writeData(w, http.StatusOK, "recent worker runs fetched successfully", totalCount, metrics)
}

func parseRunsLimit(r *http.Request) (int, error) {
	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit == "" {
		return 10, nil
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}
	if limit > 100 {
		limit = 100
	}

	return limit, nil
}
