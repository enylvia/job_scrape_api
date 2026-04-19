package handlers

import (
	"net/http"

	"job_aggregator/internal/repository"
)

type SourceHandler struct {
	sourceRepo *repository.SourceRepository
}

func NewSourceHandler(sourceRepo *repository.SourceRepository) *SourceHandler {
	return &SourceHandler{
		sourceRepo: sourceRepo,
	}
}

func (h *SourceHandler) List(w http.ResponseWriter, r *http.Request) {
	sources, err := h.sourceRepo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sources")
		return
	}

	writeData(w, http.StatusOK, sources)
}
