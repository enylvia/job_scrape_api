package handlers

import (
	"errors"
	"log"
	"net/http"

	"job_aggregator/internal/services/pipeline"
)

type WorkerHandler struct {
	logger   *log.Logger
	pipeline *pipeline.Service
}

func NewWorkerHandler(logger *log.Logger, pipelineService *pipeline.Service) *WorkerHandler {
	return &WorkerHandler{
		logger:   logger,
		pipeline: pipelineService,
	}
}

func (h *WorkerHandler) Run(w http.ResponseWriter, r *http.Request) {
	if err := h.pipeline.RunAsync(); err != nil {
		if errors.Is(err, pipeline.ErrAlreadyRunning) {
			writeError(w, http.StatusConflict, "worker pipeline is already running")
			return
		}

		h.logger.Printf("worker handler: run pipeline error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to run worker pipeline")
		return
	}

	writeData(w, http.StatusAccepted, map[string]any{
		"message": "worker pipeline started",
		"status":  h.pipeline.Status(),
	})
}

func (h *WorkerHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, h.pipeline.Status())
}
