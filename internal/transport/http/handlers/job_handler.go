package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"job_aggregator/internal/enums"
	"job_aggregator/internal/models"
	"job_aggregator/internal/repository"
)

var slugTokenPattern = regexp.MustCompile(`[^a-z0-9]+`)

type JobHandler struct {
	logger  *log.Logger
	jobRepo *repository.JobRepository
}

type updateJobRequest struct {
	SourceApplyURL         *string `json:"source_apply_url"`
	Title                  *string `json:"title"`
	Slug                   *string `json:"slug"`
	Company                *string `json:"company"`
	CompanyProfileImageURL *string `json:"company_profile_image_url"`
	Location               *string `json:"location"`
	EmploymentType         *string `json:"employment_type"`
	WorkType               *string `json:"work_type"`
	Category               *string `json:"category"`
	Description            *string `json:"description"`
	Requirements           *string `json:"requirements"`
	Benefits               *string `json:"benefits"`
	ExpiredAt              *string `json:"expired_at"`
}

func NewJobHandler(logger *log.Logger, jobRepo *repository.JobRepository) *JobHandler {
	return &JobHandler{
		logger:  logger,
		jobRepo: jobRepo,
	}
}

func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseJobListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	jobs, totalCount, err := h.jobRepo.List(r.Context(), filter)
	if err != nil {
		h.logger.Printf("job handler: list jobs error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}

	writeData(w, http.StatusOK, "jobs fetched successfully", totalCount, jobs)
}

func (h *JobHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, totalCount, err := h.jobRepo.ListCategories(r.Context())
	if err != nil {
		h.logger.Printf("job handler: list categories error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to list job categories")
		return
	}

	writeData(w, http.StatusOK, "job categories fetched successfully", totalCount, categories)
}

func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	job, err := h.jobRepo.GetByID(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}

		h.logger.Printf("job handler: get job id=%d error=%v", jobID, err)
		writeError(w, http.StatusInternalServerError, "failed to get job detail")
		return
	}

	writeData(w, http.StatusOK, "job detail fetched successfully", 1, job)
}

func (h *JobHandler) Patch(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var request updateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	job, err := h.jobRepo.GetByID(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}

		h.logger.Printf("job handler: load job for patch id=%d error=%v", jobID, err)
		writeError(w, http.StatusInternalServerError, "failed to load job")
		return
	}

	updatedJob, err := applyJobUpdate(job, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.jobRepo.UpdateEditable(r.Context(), updatedJob); err != nil {
		h.logger.Printf("job handler: patch job id=%d error=%v", jobID, err)
		writeError(w, http.StatusInternalServerError, "failed to update job")
		return
	}

	job, err = h.jobRepo.GetByID(r.Context(), jobID)
	if err != nil {
		h.logger.Printf("job handler: reload patched job id=%d error=%v", jobID, err)
		writeError(w, http.StatusInternalServerError, "failed to reload job")
		return
	}

	writeData(w, http.StatusOK, "job updated successfully", 1, job)
}

func (h *JobHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.updateJobStatus(w, r, string(enums.JobStatusApproved))
}

func (h *JobHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.updateJobStatus(w, r, string(enums.JobStatusRejected))
}

func (h *JobHandler) Archive(w http.ResponseWriter, r *http.Request) {
	h.updateJobStatus(w, r, string(enums.JobStatusArchived))
}

func (h *JobHandler) updateJobStatus(w http.ResponseWriter, r *http.Request, status string) {
	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.jobRepo.GetByID(r.Context(), jobID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}

		h.logger.Printf("job handler: load job for status update id=%d error=%v", jobID, err)
		writeError(w, http.StatusInternalServerError, "failed to load job")
		return
	}

	if err := h.jobRepo.UpdateStatus(r.Context(), jobID, status); err != nil {
		h.logger.Printf("job handler: update status id=%d status=%s error=%v", jobID, status, err)
		writeError(w, http.StatusInternalServerError, "failed to update job status")
		return
	}

	job, err := h.jobRepo.GetByID(r.Context(), jobID)
	if err != nil {
		h.logger.Printf("job handler: reload job after status update id=%d error=%v", jobID, err)
		writeError(w, http.StatusInternalServerError, "failed to reload job")
		return
	}

	writeData(w, http.StatusOK, "job status updated successfully", 1, job)
}

func parseJobID(r *http.Request) (int64, error) {
	rawID := strings.TrimSpace(r.PathValue("id"))
	if rawID == "" {
		return 0, fmt.Errorf("job id is required")
	}

	jobID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || jobID <= 0 {
		return 0, fmt.Errorf("job id must be a positive integer")
	}

	return jobID, nil
}

func parseJobListFilter(r *http.Request) (repository.JobListFilter, error) {
	query := r.URL.Query()
	filter := repository.JobListFilter{
		Status:   strings.TrimSpace(query.Get("status")),
		Category: strings.TrimSpace(query.Get("category")),
		Location: strings.TrimSpace(query.Get("location")),
		WorkType: strings.TrimSpace(query.Get("work_type")),
		RoleType: strings.TrimSpace(query.Get("role_type")),
		Sort:     strings.TrimSpace(query.Get("sort")),
	}

	if rawSearch := strings.TrimSpace(query.Get("search")); rawSearch != "" {
		filter.Search = rawSearch
	}
	if rawSourceID := strings.TrimSpace(query.Get("source_id")); rawSourceID != "" {
		sourceID, err := strconv.ParseInt(rawSourceID, 10, 64)
		if err != nil || sourceID <= 0 {
			return repository.JobListFilter{}, fmt.Errorf("source_id must be a positive integer")
		}
		filter.SourceID = &sourceID
	}

	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return repository.JobListFilter{}, fmt.Errorf("limit must be a positive integer")
		}
		filter.Limit = limit
	}

	if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return repository.JobListFilter{}, fmt.Errorf("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}

	if rawSort := strings.TrimSpace(query.Get("sort")); rawSort != "" && !strings.EqualFold(rawSort, "asc") && !strings.EqualFold(rawSort, "desc") {
		return repository.JobListFilter{}, fmt.Errorf("sort must be asc or desc")
	}

	if rawCreatedFrom := strings.TrimSpace(query.Get("created_from")); rawCreatedFrom != "" {
		createdFrom, err := parseDateTime(rawCreatedFrom)
		if err != nil {
			return repository.JobListFilter{}, fmt.Errorf("created_from must be RFC3339 or YYYY-MM-DD")
		}
		filter.CreatedFrom = &createdFrom
	}

	if rawCreatedTo := strings.TrimSpace(query.Get("created_to")); rawCreatedTo != "" {
		createdTo, err := parseDateTime(rawCreatedTo)
		if err != nil {
			return repository.JobListFilter{}, fmt.Errorf("created_to must be RFC3339 or YYYY-MM-DD")
		}
		if len(rawCreatedTo) == len("2006-01-02") {
			createdTo = createdTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		filter.CreatedTo = &createdTo
	}

	return filter, nil
}

func parseDateTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid datetime")
}

func applyJobUpdate(job models.Job, request updateJobRequest) (models.Job, error) {
	if request.SourceApplyURL != nil {
		job.SourceApplyURL = strings.TrimSpace(*request.SourceApplyURL)
	}

	titleChanged := false
	if request.Title != nil {
		job.Title = strings.TrimSpace(*request.Title)
		titleChanged = true
	}

	if request.Slug != nil {
		job.Slug = slugify(*request.Slug)
	} else if titleChanged {
		job.Slug = slugify(job.Title)
	}

	if request.Company != nil {
		job.Company = strings.TrimSpace(*request.Company)
	}
	if request.CompanyProfileImageURL != nil {
		job.CompanyProfileImageURL = strings.TrimSpace(*request.CompanyProfileImageURL)
	}
	if request.Location != nil {
		job.Location = strings.TrimSpace(*request.Location)
	}
	if request.EmploymentType != nil {
		job.EmploymentType = strings.TrimSpace(*request.EmploymentType)
	}
	if request.WorkType != nil {
		job.WorkType = strings.TrimSpace(*request.WorkType)
	}
	if request.Category != nil {
		job.Category = strings.TrimSpace(*request.Category)
	}
	if request.Description != nil {
		job.Description = strings.TrimSpace(*request.Description)
	}
	if request.Requirements != nil {
		job.Requirements = strings.TrimSpace(*request.Requirements)
	}
	if request.Benefits != nil {
		job.Benefits = strings.TrimSpace(*request.Benefits)
	}
	if request.ExpiredAt != nil {
		rawExpiredAt := strings.TrimSpace(*request.ExpiredAt)
		if rawExpiredAt == "" {
			job.ExpiredAt = nil
		} else {
			expiredAt, err := time.Parse(time.RFC3339, rawExpiredAt)
			if err != nil {
				return models.Job{}, fmt.Errorf("expired_at must be RFC3339")
			}
			job.ExpiredAt = &expiredAt
		}
	}

	return job, nil
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	value = slugTokenPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}
