package models

import (
	"encoding/json"
	"time"
)

type AnalyticsEvent struct {
	ID         int64           `json:"id"`
	EventName  string          `json:"event_name"`
	VisitorID  string          `json:"visitor_id"`
	SessionID  string          `json:"session_id"`
	Path       string          `json:"path"`
	JobID      *int64          `json:"job_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata"`
	UserAgent  string          `json:"user_agent"`
	IPHash     string          `json:"-"`
	OccurredAt time.Time       `json:"occurred_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

type AnalyticsSummary struct {
	WindowStartedAt   time.Time          `json:"window_started_at"`
	WindowFinishedAt  time.Time          `json:"window_finished_at"`
	VisitorsToday     int                `json:"visitors_today"`
	PageViewsToday    int                `json:"page_views_today"`
	JobViewsToday     int                `json:"job_views_today"`
	ApplyClicksToday  int                `json:"apply_clicks_today"`
	SearchesToday     int                `json:"searches_today"`
	ConversionRate    float64            `json:"conversion_rate"`
	TopViewedJobs     []TopViewedJob     `json:"top_viewed_jobs"`
	TopSearchKeywords []TopSearchKeyword `json:"top_search_keywords"`
}

type TopViewedJob struct {
	JobID     int64  `json:"job_id"`
	Title     string `json:"title"`
	Company   string `json:"company"`
	ViewCount int    `json:"view_count"`
}

type TopSearchKeyword struct {
	Keyword     string `json:"keyword"`
	SearchCount int    `json:"search_count"`
}
