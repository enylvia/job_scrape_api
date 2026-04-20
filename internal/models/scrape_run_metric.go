package models

import "time"

type ScrapeRunMetric struct {
	ID                     int64     `json:"id"`
	StartedAt              time.Time `json:"started_at"`
	FinishedAt             time.Time `json:"finished_at"`
	DurationSeconds        int64     `json:"duration_seconds"`
	TotalSources           int       `json:"total_sources"`
	SuccessfulSources      int       `json:"successful_sources"`
	FailedSources          int       `json:"failed_sources"`
	TotalJobsCollected     int       `json:"total_jobs_collected"`
	SavedJobs              int       `json:"saved_jobs"`
	SkippedJobs            int       `json:"skipped_jobs"`
	SuccessRatePercentage  float64   `json:"success_rate_percentage"`
	ScrapeHealthPercentage float64   `json:"scrape_health_percentage"`
	CreatedAt              time.Time `json:"created_at"`
}

type ScrapeRunMetricSummary struct {
	WindowStartedAt        time.Time `json:"window_started_at"`
	WindowFinishedAt       time.Time `json:"window_finished_at"`
	TotalRuns              int       `json:"total_runs"`
	TotalSources           int       `json:"total_sources"`
	SuccessfulSources      int       `json:"successful_sources"`
	FailedSources          int       `json:"failed_sources"`
	TotalJobsCollected     int       `json:"total_jobs_collected"`
	TotalSavedJobs         int       `json:"total_saved_jobs"`
	TotalSkippedJobs       int       `json:"total_skipped_jobs"`
	AvgRunDurationSeconds  float64   `json:"avg_run_duration_seconds"`
	SuccessRatePercentage  float64   `json:"success_rate_percentage"`
	ScrapeHealthPercentage float64   `json:"scrape_health_percentage"`
}
