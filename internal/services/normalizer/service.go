package normalizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"job_aggregator/internal/enums"
	"job_aggregator/internal/models"
	"job_aggregator/internal/repository"
)

type Service struct {
	logger  *log.Logger
	jobRepo *repository.JobRepository
}

func NewService(logger *log.Logger, jobRepo *repository.JobRepository) *Service {
	return &Service{
		logger:  logger,
		jobRepo: jobRepo,
	}
}

func (s *Service) RunOnce(ctx context.Context) error {
	jobs, err := s.jobRepo.ListByStatus(ctx, string(enums.JobStatusScraped))
	if err != nil {
		return fmt.Errorf("list scraped jobs: %w", err)
	}

	if len(jobs) == 0 {
		s.logger.Println("normalizer worker: no scraped jobs to process")
		return nil
	}

	normalizedCount := 0
	reviewPendingCount := 0
	for _, job := range jobs {
		normalized := normalizeJob(job)
		if err := s.jobRepo.UpdateNormalized(ctx, normalized); err != nil {
			s.logger.Printf("normalizer worker: job_id=%d source_job_url=%s update error=%v", job.ID, job.SourceJobURL, err)
			continue
		}

		switch normalized.Status {
		case string(enums.JobStatusNormalized):
			normalizedCount++
		case string(enums.JobStatusReviewPending):
			reviewPendingCount++
		}
	}

	s.logger.Printf("normalizer worker: processed=%d normalized=%d review_pending=%d", len(jobs), normalizedCount, reviewPendingCount)
	return nil
}

func normalizeJob(job models.Job) models.Job {
	job.SourceJobURL = cleanSingleLine(job.SourceJobURL)
	job.SourceApplyURL = cleanSingleLine(job.SourceApplyURL)
	job.Title = cleanSingleLine(job.Title)
	job.Company = cleanSingleLine(job.Company)
	job.Location = normalizeLocation(job.Location)
	job.EmploymentType = normalizeEmploymentType(job.EmploymentType)
	job.Category = cleanSingleLine(job.Category)
	job.Description = cleanMultiline(job.Description)
	job.Requirements = cleanMultiline(job.Requirements)
	job.Benefits = cleanMultiline(job.Benefits)
	job.Slug = generateSlug(firstNonEmpty(job.Title, job.Slug))

	if isValidNormalizedJob(job) {
		job.Status = string(enums.JobStatusNormalized)
	} else {
		job.Status = string(enums.JobStatusReviewPending)
	}

	job.ContentHash = buildContentHash(job)
	return job
}

func isValidNormalizedJob(job models.Job) bool {
	required := []string{
		job.SourceJobURL,
		job.Title,
		job.Company,
		job.Location,
		job.EmploymentType,
		job.Description,
		job.Slug,
	}

	for _, value := range required {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}

	return true
}

func buildContentHash(job models.Job) string {
	value := strings.Join([]string{
		job.SourceJobURL,
		job.SourceApplyURL,
		job.Title,
		job.Slug,
		job.Company,
		job.Location,
		job.EmploymentType,
		job.Category,
		formatNullableInt64(job.SalaryMin),
		formatNullableInt64(job.SalaryMax),
		job.Currency,
		job.Description,
		job.Requirements,
		job.Benefits,
	}, "|")

	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

func formatNullableInt64(value *int64) string {
	if value == nil {
		return ""
	}

	return fmt.Sprintf("%d", *value)
}
