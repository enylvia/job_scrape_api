package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"job_aggregator/internal/models"
)

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) CreateEvent(ctx context.Context, event models.AnalyticsEvent) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database is not configured")
	}

	metadata := event.Metadata
	if !json.Valid(metadata) {
		metadata = json.RawMessage(`{}`)
	}

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO analytics_events (
			event_name,
			visitor_id,
			session_id,
			path,
			job_id,
			metadata,
			user_agent,
			ip_hash,
			occurred_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)
		RETURNING id
	`,
		event.EventName,
		event.VisitorID,
		event.SessionID,
		event.Path,
		event.JobID,
		string(metadata),
		event.UserAgent,
		event.IPHash,
		event.OccurredAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert analytics event: %w", err)
	}

	return id, nil
}

func (r *AnalyticsRepository) GetSummary(ctx context.Context, now time.Time, topLimit int) (models.AnalyticsSummary, error) {
	if r.db == nil {
		return models.AnalyticsSummary{}, fmt.Errorf("database is not configured")
	}

	if topLimit <= 0 {
		topLimit = 5
	}
	if topLimit > 50 {
		topLimit = 50
	}

	windowFinishedAt := now
	windowStartedAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	summary := models.AnalyticsSummary{
		WindowStartedAt:  windowStartedAt,
		WindowFinishedAt: windowFinishedAt,
	}

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(DISTINCT NULLIF(visitor_id, '')) AS visitors_today,
			COUNT(*) FILTER (WHERE event_name = 'page_view') AS page_views_today,
			COUNT(*) FILTER (WHERE event_name = 'job_view') AS job_views_today,
			COUNT(*) FILTER (WHERE event_name = 'apply_clicked') AS apply_clicks_today,
			COUNT(*) FILTER (WHERE event_name = 'search_performed') AS searches_today
		FROM analytics_events
		WHERE occurred_at >= $1 AND occurred_at <= $2
	`, windowStartedAt, windowFinishedAt).Scan(
		&summary.VisitorsToday,
		&summary.PageViewsToday,
		&summary.JobViewsToday,
		&summary.ApplyClicksToday,
		&summary.SearchesToday,
	)
	if err != nil {
		return models.AnalyticsSummary{}, fmt.Errorf("get analytics summary counts: %w", err)
	}

	summary.ConversionRate = percentage(summary.ApplyClicksToday, summary.JobViewsToday)

	topViewedJobs, err := r.listTopViewedJobs(ctx, windowStartedAt, windowFinishedAt, topLimit)
	if err != nil {
		return models.AnalyticsSummary{}, err
	}
	summary.TopViewedJobs = topViewedJobs

	topSearchKeywords, err := r.listTopSearchKeywords(ctx, windowStartedAt, windowFinishedAt, topLimit)
	if err != nil {
		return models.AnalyticsSummary{}, err
	}
	summary.TopSearchKeywords = topSearchKeywords

	return summary, nil
}

func (r *AnalyticsRepository) listTopViewedJobs(ctx context.Context, startedAt, finishedAt time.Time, limit int) ([]models.TopViewedJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			j.id,
			j.title,
			j.company,
			COUNT(*) AS view_count
		FROM analytics_events ae
		JOIN jobs j ON j.id = ae.job_id
		WHERE ae.event_name = 'job_view'
			AND ae.job_id IS NOT NULL
			AND ae.occurred_at >= $1
			AND ae.occurred_at <= $2
		GROUP BY j.id, j.title, j.company
		ORDER BY view_count DESC, j.id DESC
		LIMIT $3
	`, startedAt, finishedAt, limit)
	if err != nil {
		return nil, fmt.Errorf("query top viewed jobs: %w", err)
	}
	defer rows.Close()

	items := make([]models.TopViewedJob, 0)
	for rows.Next() {
		var item models.TopViewedJob
		if err := rows.Scan(&item.JobID, &item.Title, &item.Company, &item.ViewCount); err != nil {
			return nil, fmt.Errorf("scan top viewed job: %w", err)
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top viewed jobs: %w", err)
	}

	return items, nil
}

func (r *AnalyticsRepository) listTopSearchKeywords(ctx context.Context, startedAt, finishedAt time.Time, limit int) ([]models.TopSearchKeyword, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			LOWER(TRIM(metadata->>'keyword')) AS keyword,
			COUNT(*) AS search_count
		FROM analytics_events
		WHERE event_name = 'search_performed'
			AND occurred_at >= $1
			AND occurred_at <= $2
			AND NULLIF(TRIM(metadata->>'keyword'), '') IS NOT NULL
		GROUP BY LOWER(TRIM(metadata->>'keyword'))
		ORDER BY search_count DESC, keyword ASC
		LIMIT $3
	`, startedAt, finishedAt, limit)
	if err != nil {
		return nil, fmt.Errorf("query top search keywords: %w", err)
	}
	defer rows.Close()

	items := make([]models.TopSearchKeyword, 0)
	for rows.Next() {
		var item models.TopSearchKeyword
		if err := rows.Scan(&item.Keyword, &item.SearchCount); err != nil {
			return nil, fmt.Errorf("scan top search keyword: %w", err)
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top search keywords: %w", err)
	}

	return items, nil
}

func percentage(part, total int) float64 {
	if total <= 0 {
		return 0
	}

	return math.Round((float64(part)/float64(total))*10000) / 100
}
