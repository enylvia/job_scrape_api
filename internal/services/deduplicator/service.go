package deduplicator

import (
	"context"
	"fmt"
	"log"

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
	jobs, err := s.jobRepo.ListByStatus(ctx, string(enums.JobStatusNormalized))
	if err != nil {
		return fmt.Errorf("list normalized jobs: %w", err)
	}

	if len(jobs) == 0 {
		s.logger.Println("deduplicator worker: no normalized jobs to process")
		return nil
	}

	canonicals := make([]models.Job, 0, len(jobs))
	duplicateCount := 0

	for _, job := range jobs {
		matched, duplicateOfID, reason := findDuplicate(job, canonicals)
		if !matched {
			canonicals = append(canonicals, job)
			continue
		}

		if err := s.jobRepo.MarkDuplicate(ctx, job.ID, duplicateOfID); err != nil {
			s.logger.Printf("deduplicator worker: job_id=%d duplicate_of=%d update error=%v", job.ID, duplicateOfID, err)
			canonicals = append(canonicals, job)
			continue
		}

		duplicateCount++
		s.logger.Printf("deduplicator worker: job_id=%d duplicate_of=%d reason=%s", job.ID, duplicateOfID, reason)
	}

	s.logger.Printf("deduplicator worker: processed=%d duplicates=%d canonical=%d", len(jobs), duplicateCount, len(canonicals))
	return nil
}

func findDuplicate(candidate models.Job, canonicals []models.Job) (bool, int64, string) {
	for _, existing := range canonicals {
		if matched, reason := detectExactDuplicate(candidate, existing); matched {
			primary := duplicatePrimary(candidate, existing)
			if candidate.ID == primary.ID {
				return false, 0, ""
			}
			return true, primary.ID, reason
		}
	}

	for _, existing := range canonicals {
		matched, _, reason := detectFuzzyDuplicate(candidate, existing)
		if !matched {
			continue
		}

		primary := duplicatePrimary(candidate, existing)
		if candidate.ID == primary.ID {
			return false, 0, ""
		}
		return true, primary.ID, reason
	}

	return false, 0, ""
}
