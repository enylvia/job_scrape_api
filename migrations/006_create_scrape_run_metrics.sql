CREATE TABLE IF NOT EXISTS scrape_run_metrics (
    id BIGSERIAL PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    duration_seconds BIGINT NOT NULL DEFAULT 0,
    total_sources INTEGER NOT NULL DEFAULT 0,
    successful_sources INTEGER NOT NULL DEFAULT 0,
    failed_sources INTEGER NOT NULL DEFAULT 0,
    total_jobs_collected INTEGER NOT NULL DEFAULT 0,
    saved_jobs INTEGER NOT NULL DEFAULT 0,
    skipped_jobs INTEGER NOT NULL DEFAULT 0,
    success_rate_percentage NUMERIC(5,2) NOT NULL DEFAULT 0,
    scrape_health_percentage NUMERIC(5,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scrape_run_metrics_started_at
    ON scrape_run_metrics(started_at DESC);
