package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"job_aggregator/internal/models"
)

type ScrapeRunMetricRepository struct {
	db *sql.DB
}

func NewScrapeRunMetricRepository(db *sql.DB) *ScrapeRunMetricRepository {
	return &ScrapeRunMetricRepository{db: db}
}

func (r *ScrapeRunMetricRepository) Create(ctx context.Context, metric models.ScrapeRunMetric) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database is not configured")
	}

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO scrape_run_metrics (
			started_at,
			finished_at,
			duration_seconds,
			total_sources,
			successful_sources,
			failed_sources,
			total_jobs_collected,
			saved_jobs,
			skipped_jobs,
			success_rate_percentage,
			scrape_health_percentage
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`,
		metric.StartedAt,
		metric.FinishedAt,
		metric.DurationSeconds,
		metric.TotalSources,
		metric.SuccessfulSources,
		metric.FailedSources,
		metric.TotalJobsCollected,
		metric.SavedJobs,
		metric.SkippedJobs,
		metric.SuccessRatePercentage,
		metric.ScrapeHealthPercentage,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert scrape run metric: %w", err)
	}

	return id, nil
}

func (r *ScrapeRunMetricRepository) Get24hSummary(ctx context.Context, now time.Time) (models.ScrapeRunMetricSummary, error) {
	if r.db == nil {
		return models.ScrapeRunMetricSummary{}, fmt.Errorf("database is not configured")
	}

	windowFinishedAt := now.UTC()
	windowStartedAt := windowFinishedAt.Add(-24 * time.Hour)

	summary := models.ScrapeRunMetricSummary{
		WindowStartedAt:  windowStartedAt,
		WindowFinishedAt: windowFinishedAt,
	}

	var avgRunDurationSeconds sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total_runs,
			COALESCE(SUM(total_sources), 0) AS total_sources,
			COALESCE(SUM(successful_sources), 0) AS successful_sources,
			COALESCE(SUM(failed_sources), 0) AS failed_sources,
			COALESCE(SUM(total_jobs_collected), 0) AS total_jobs_collected,
			COALESCE(SUM(saved_jobs), 0) AS total_saved_jobs,
			COALESCE(SUM(skipped_jobs), 0) AS total_skipped_jobs,
			AVG(duration_seconds)::float8 AS avg_run_duration_seconds
		FROM scrape_run_metrics
		WHERE started_at >= $1 AND started_at <= $2
	`, windowStartedAt, windowFinishedAt).Scan(
		&summary.TotalRuns,
		&summary.TotalSources,
		&summary.SuccessfulSources,
		&summary.FailedSources,
		&summary.TotalJobsCollected,
		&summary.TotalSavedJobs,
		&summary.TotalSkippedJobs,
		&avgRunDurationSeconds,
	)
	if err != nil {
		return models.ScrapeRunMetricSummary{}, fmt.Errorf("get 24h scrape run summary: %w", err)
	}

	if avgRunDurationSeconds.Valid {
		summary.AvgRunDurationSeconds = roundFloat(avgRunDurationSeconds.Float64)
	}

	summary.SuccessRatePercentage = calculatePercentage(summary.SuccessfulSources, summary.TotalSources)
	summary.ScrapeHealthPercentage = summary.SuccessRatePercentage

	return summary, nil
}

func (r *ScrapeRunMetricRepository) ListRecent(ctx context.Context, limit int) ([]models.ScrapeRunMetric, int, error) {
	if r.db == nil {
		return []models.ScrapeRunMetric{}, 0, nil
	}

	totalCount, err := r.countAll(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			started_at,
			finished_at,
			duration_seconds,
			total_sources,
			successful_sources,
			failed_sources,
			total_jobs_collected,
			saved_jobs,
			skipped_jobs,
			success_rate_percentage,
			scrape_health_percentage,
			created_at
		FROM scrape_run_metrics
		ORDER BY started_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list recent scrape run metrics: %w", err)
	}
	defer rows.Close()

	metrics := make([]models.ScrapeRunMetric, 0)
	for rows.Next() {
		var metric models.ScrapeRunMetric
		if err := rows.Scan(
			&metric.ID,
			&metric.StartedAt,
			&metric.FinishedAt,
			&metric.DurationSeconds,
			&metric.TotalSources,
			&metric.SuccessfulSources,
			&metric.FailedSources,
			&metric.TotalJobsCollected,
			&metric.SavedJobs,
			&metric.SkippedJobs,
			&metric.SuccessRatePercentage,
			&metric.ScrapeHealthPercentage,
			&metric.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan recent scrape run metric: %w", err)
		}

		metrics = append(metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate recent scrape run metrics: %w", err)
	}

	return metrics, totalCount, nil
}

func calculatePercentage(successCount, totalCount int) float64 {
	if totalCount <= 0 {
		return 0
	}

	percentage := (float64(successCount) / float64(totalCount)) * 100
	return roundFloat(percentage)
}

func roundFloat(value float64) float64 {
	return math.Round(value*100) / 100
}

func (r *ScrapeRunMetricRepository) countAll(ctx context.Context) (int, error) {
	var totalCount int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scrape_run_metrics`).Scan(&totalCount); err != nil {
		return 0, fmt.Errorf("count scrape run metrics: %w", err)
	}

	return totalCount, nil
}
