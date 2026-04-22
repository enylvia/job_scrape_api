package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"job_aggregator/internal/models"
	"job_aggregator/internal/repository"
)

const maxAnalyticsRequestBodyBytes = 1 << 20

var allowedAnalyticsEvents = map[string]struct{}{
	"page_view":           {},
	"job_view":            {},
	"search_performed":    {},
	"filter_used":         {},
	"apply_clicked":       {},
	"category_clicked":    {},
	"newsletter_interest": {},
}

type AnalyticsHandler struct {
	logger        *log.Logger
	analyticsRepo *repository.AnalyticsRepository
}

type trackAnalyticsEventRequest struct {
	EventName  string          `json:"event_name"`
	VisitorID  string          `json:"visitor_id"`
	SessionID  string          `json:"session_id"`
	Path       string          `json:"path"`
	JobID      *int64          `json:"job_id"`
	Metadata   json.RawMessage `json:"metadata"`
	OccurredAt *time.Time      `json:"occurred_at"`
}

type trackAnalyticsEventResponse struct {
	ID int64 `json:"id"`
}

func NewAnalyticsHandler(logger *log.Logger, analyticsRepo *repository.AnalyticsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{
		logger:        logger,
		analyticsRepo: analyticsRepo,
	}
}

func (h *AnalyticsHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAnalyticsRequestBodyBytes)

	var request trackAnalyticsEventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	event, err := h.buildAnalyticsEvent(r, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	eventID, err := h.analyticsRepo.CreateEvent(r.Context(), event)
	if err != nil {
		h.logger.Printf("analytics handler: track event error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to track analytics event")
		return
	}

	writeData(w, http.StatusCreated, "analytics event tracked successfully", 1, trackAnalyticsEventResponse{
		ID: eventID,
	})
}

func (h *AnalyticsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	limit, err := parseAnalyticsTopLimit(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	summary, err := h.analyticsRepo.GetSummary(r.Context(), time.Now(), limit)
	if err != nil {
		h.logger.Printf("analytics handler: get summary error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to get analytics summary")
		return
	}

	writeData(w, http.StatusOK, "analytics summary fetched successfully", 1, summary)
}

func (h *AnalyticsHandler) buildAnalyticsEvent(r *http.Request, request trackAnalyticsEventRequest) (models.AnalyticsEvent, error) {
	eventName := strings.TrimSpace(request.EventName)
	if _, ok := allowedAnalyticsEvents[eventName]; !ok {
		return models.AnalyticsEvent{}, fmt.Errorf("event_name is invalid")
	}

	visitorID := strings.TrimSpace(request.VisitorID)
	if visitorID == "" {
		return models.AnalyticsEvent{}, fmt.Errorf("visitor_id is required")
	}

	sessionID := strings.TrimSpace(request.SessionID)
	path := strings.TrimSpace(request.Path)
	if path == "" {
		path = r.URL.Path
	}
	if len(path) > 2048 {
		return models.AnalyticsEvent{}, fmt.Errorf("path is too long")
	}

	if (eventName == "job_view" || eventName == "apply_clicked") && (request.JobID == nil || *request.JobID <= 0) {
		return models.AnalyticsEvent{}, fmt.Errorf("job_id is required for %s", eventName)
	}

	metadata := request.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	if !json.Valid(metadata) {
		return models.AnalyticsEvent{}, fmt.Errorf("metadata must be valid JSON")
	}

	occurredAt := time.Now()
	if request.OccurredAt != nil {
		occurredAt = *request.OccurredAt
	}

	return models.AnalyticsEvent{
		EventName:  eventName,
		VisitorID:  visitorID,
		SessionID:  sessionID,
		Path:       path,
		JobID:      request.JobID,
		Metadata:   metadata,
		UserAgent:  strings.TrimSpace(r.UserAgent()),
		IPHash:     hashIP(clientIPFromRequest(r)),
		OccurredAt: occurredAt,
	}, nil
}

func parseAnalyticsTopLimit(r *http.Request) (int, error) {
	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit == "" {
		return 5, nil
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}
	if limit > 50 {
		limit = 50
	}

	return limit, nil
}

func clientIPFromRequest(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		ip, _, _ := strings.Cut(forwardedFor, ",")
		return strings.TrimSpace(ip)
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

func hashIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}
