package collector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"job_aggregator/internal/enums"
	"job_aggregator/internal/models"
	"job_aggregator/internal/repository"
)

type Service struct {
	logger         *log.Logger
	sourceRepo     *repository.SourceRepository
	jobRepo        *repository.JobRepository
	jobRawDataRepo *repository.JobRawDataRepository
	collectors     map[string]Collector
	scrapers       map[string]SourceScraper
}

type RunSummary struct {
	TotalSources       int
	SuccessfulSources  int
	FailedSources      int
	TotalJobsCollected int
	SavedJobs          int
	SkippedJobs        int
}

type SourceRunSummary struct {
	JobsCollected int
	SavedJobs     int
	SkippedJobs   int
}

func NewService(
	logger *log.Logger,
	sourceRepo *repository.SourceRepository,
	jobRepo *repository.JobRepository,
	jobRawDataRepo *repository.JobRawDataRepository,
	collectors []Collector,
	scrapers []SourceScraper,
) *Service {
	collectorMap := make(map[string]Collector, len(collectors))
	for _, item := range collectors {
		collectorMap[item.Mode()] = item
	}

	scraperMap := make(map[string]SourceScraper, len(scrapers))
	for _, item := range scrapers {
		scraperMap[item.Name()] = item
	}

	return &Service{
		logger:         logger,
		sourceRepo:     sourceRepo,
		jobRepo:        jobRepo,
		jobRawDataRepo: jobRawDataRepo,
		collectors:     collectorMap,
		scrapers:       scraperMap,
	}
}

func (s *Service) RunOnce(ctx context.Context) (RunSummary, error) {
	summary := RunSummary{}

	sources, err := s.sourceRepo.ListActive(ctx)
	if err != nil {
		return summary, fmt.Errorf("list active sources: %w", err)
	}
	summary.TotalSources = len(sources)

	if len(sources) == 0 {
		s.logger.Println("collector worker: no active sources to process")
		return summary, nil
	}

	for _, source := range sources {
		sourceSummary, err := s.collectSource(ctx, source)
		if err != nil {
			summary.FailedSources++
			s.logger.Printf("collector worker: source=%s mode=%s error=%v", source.Name, source.Mode, err)
			continue
		}

		summary.SuccessfulSources++
		summary.TotalJobsCollected += sourceSummary.JobsCollected
		summary.SavedJobs += sourceSummary.SavedJobs
		summary.SkippedJobs += sourceSummary.SkippedJobs
	}

	return summary, nil
}

func (s *Service) collectSource(ctx context.Context, source models.Source) (SourceRunSummary, error) {
	summary := SourceRunSummary{}

	scraper, ok := s.scrapers[source.Name]
	if !ok {
		return summary, fmt.Errorf("no scraper registered for source %q", source.Name)
	}

	engine, ok := s.collectors[source.Mode]
	if !ok {
		return summary, fmt.Errorf("no collector registered for mode %q", source.Mode)
	}

	jobs, err := engine.Collect(ctx, source, scraper)
	if err != nil {
		return summary, fmt.Errorf("collect jobs: %w", err)
	}
	summary.JobsCollected = len(jobs)

	savedCount := 0
	skippedCount := 0
	seenURLs := make(map[string]struct{})
	for _, item := range jobs {
		if item.SourceJobURL != "" {
			if _, seen := seenURLs[item.SourceJobURL]; seen {
				s.logger.Printf("collector worker: source=%s job=%s skipped duplicate in current batch", source.Name, item.SourceJobURL)
				skippedCount++
				continue
			}
			seenURLs[item.SourceJobURL] = struct{}{}
		}

		saved, err := s.persistCollectedJob(ctx, source.ID, item)
		if err != nil {
			s.logger.Printf("collector worker: source=%s job=%s persist error=%v", source.Name, item.SourceJobURL, err)
			continue
		}
		if saved {
			savedCount++
			continue
		}

		skippedCount++
	}

	if err := s.sourceRepo.MarkScraped(ctx, source.ID, time.Now().UTC()); err != nil {
		s.logger.Printf("collector worker: source=%s update last_scraped_at error=%v", source.Name, err)
	}

	summary.SavedJobs = savedCount
	summary.SkippedJobs = skippedCount
	s.logger.Printf("collector worker: source=%s collected=%d saved=%d skipped=%d", source.Name, len(jobs), savedCount, skippedCount)
	return summary, nil
}

func (s *Service) persistCollectedJob(ctx context.Context, sourceID int64, item CollectedJob) (bool, error) {
	collectedAt := item.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}

	exists, err := s.jobRepo.ExistsBySourceJobURL(ctx, sourceID, item.SourceJobURL)
	if err != nil {
		return false, fmt.Errorf("check existing job: %w", err)
	}
	if exists {
		return false, nil
	}

	jobID, err := s.jobRepo.Create(ctx, models.Job{
		SourceID:       sourceID,
		SourceJobURL:   item.SourceJobURL,
		SourceApplyURL: item.SourceApplyURL,
		Title:          item.Title,
		Slug:           item.Slug,
		Company:        item.Company,
		Location:       item.Location,
		EmploymentType: item.EmploymentType,
		Category:       item.Category,
		SalaryMin:      item.SalaryMin,
		SalaryMax:      item.SalaryMax,
		Currency:       item.Currency,
		Description:    item.Description,
		Requirements:   item.Requirements,
		Benefits:       item.Benefits,
		PostedAt:       item.PostedAt,
		ExpiredAt:      item.ExpiredAt,
		ContentHash:    buildContentHash(item),
		Status:         string(enums.JobStatusScraped),
		TelegramSent:   false,
	})
	if err != nil {
		return false, fmt.Errorf("create job: %w", err)
	}

	if strings.TrimSpace(item.RawHTML) == "" && strings.TrimSpace(item.RawJSON) == "" {
		return true, nil
	}

	_, err = s.jobRawDataRepo.Create(ctx, models.JobRawData{
		JobID:     jobID,
		RawHTML:   item.RawHTML,
		RawJSON:   item.RawJSON,
		ScrapedAt: collectedAt,
	})
	if err != nil {
		return false, fmt.Errorf("create job raw data: %w", err)
	}

	return true, nil
}

func buildContentHash(item CollectedJob) string {
	value := strings.Join([]string{
		item.SourceJobURL,
		item.SourceApplyURL,
		item.Title,
		item.Slug,
		item.Company,
		item.Location,
		item.EmploymentType,
		item.WorkplaceType,
		item.Category,
		formatNullableInt64(item.SalaryMin),
		formatNullableInt64(item.SalaryMax),
		item.Currency,
		item.Description,
		item.Requirements,
		item.Benefits,
	}, "|")

	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func formatNullableInt64(value *int64) string {
	if value == nil {
		return ""
	}

	return fmt.Sprintf("%d", *value)
}
