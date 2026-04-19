package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"job_aggregator/internal/repository"
	"job_aggregator/internal/services/collector"
	"job_aggregator/internal/services/collector/browsercollector"
	"job_aggregator/internal/services/collector/httpcollector"
	"job_aggregator/internal/services/collector/sources"
	"job_aggregator/internal/services/deduplicator"
	"job_aggregator/internal/services/normalizer"
)

var ErrAlreadyRunning = errors.New("worker pipeline is already running")

type Result struct {
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt time.Time     `json:"finished_at"`
	Duration   time.Duration `json:"duration"`
}

type Status struct {
	Running        bool       `json:"running"`
	LastStartedAt  *time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt *time.Time `json:"last_finished_at,omitempty"`
	LastDuration   string     `json:"last_duration,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
}

type Service struct {
	logger       *log.Logger
	collector    *collector.Service
	normalizer   *normalizer.Service
	deduplicator *deduplicator.Service
	running      atomic.Bool
	mu           sync.RWMutex
	lastResult   *Result
	lastError    string
}

func NewService(
	logger *log.Logger,
	sourceRepo *repository.SourceRepository,
	jobRepo *repository.JobRepository,
	jobRawDataRepo *repository.JobRawDataRepository,
) *Service {
	return &Service{
		logger: logger,
		collector: collector.NewService(
			logger,
			sourceRepo,
			jobRepo,
			jobRawDataRepo,
			[]collector.Collector{
				httpcollector.New(),
				browsercollector.New(),
			},
			[]collector.SourceScraper{
				sources.NewDeallsScraper(),
				sources.NewGlintsScraper(),
			},
		),
		normalizer:   normalizer.NewService(logger, jobRepo),
		deduplicator: deduplicator.NewService(logger, jobRepo),
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if !s.running.CompareAndSwap(false, true) {
		return Result{}, ErrAlreadyRunning
	}
	return s.run(ctx)
}

func (s *Service) RunAsync() error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyRunning
	}

	go func() {
		if _, err := s.run(context.Background()); err != nil && !errors.Is(err, ErrAlreadyRunning) {
			s.logger.Printf("pipeline worker: async run error=%v", err)
		}
	}()

	return nil
}

func (s *Service) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := Status{
		Running:   s.running.Load(),
		LastError: s.lastError,
	}

	if s.lastResult != nil {
		startedAt := s.lastResult.StartedAt
		status.LastStartedAt = &startedAt

		if !s.lastResult.FinishedAt.IsZero() {
			finishedAt := s.lastResult.FinishedAt
			status.LastFinishedAt = &finishedAt
			status.LastDuration = s.lastResult.Duration.String()
		}
	}

	return status
}

func (s *Service) setLastResult(result Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resultCopy := result
	s.lastResult = &resultCopy
	s.lastError = ""
}

func (s *Service) setLastError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastError = err.Error()
}

func (s *Service) clearLastError() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastError = ""
}

func (s *Service) setLastStartedAt(startedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastResult == nil {
		s.lastResult = &Result{}
	}
	s.lastResult.StartedAt = startedAt
	s.lastResult.FinishedAt = time.Time{}
	s.lastResult.Duration = 0
}

func (s *Service) run(ctx context.Context) (Result, error) {
	defer s.running.Store(false)

	s.clearLastError()
	startedAt := time.Now().UTC()
	s.setLastStartedAt(startedAt)
	s.logger.Printf("pipeline worker: started at=%s", startedAt.Format(time.RFC3339))

	if err := s.collector.RunOnce(ctx); err != nil {
		s.setLastError(err)
		return Result{}, fmt.Errorf("run collector: %w", err)
	}

	if err := s.normalizer.RunOnce(ctx); err != nil {
		s.setLastError(err)
		return Result{}, fmt.Errorf("run normalizer: %w", err)
	}

	if err := s.deduplicator.RunOnce(ctx); err != nil {
		s.setLastError(err)
		return Result{}, fmt.Errorf("run deduplicator: %w", err)
	}

	finishedAt := time.Now().UTC()
	result := Result{
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Duration:   finishedAt.Sub(startedAt),
	}

	s.setLastResult(result)
	s.logger.Printf("pipeline worker: completed duration=%s", result.Duration)
	return result, nil
}
