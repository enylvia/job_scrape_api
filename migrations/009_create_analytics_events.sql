CREATE TABLE IF NOT EXISTS analytics_events (
    id BIGSERIAL PRIMARY KEY,
    event_name TEXT NOT NULL,
    visitor_id TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    job_id BIGINT NULL REFERENCES jobs(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    user_agent TEXT NOT NULL DEFAULT '',
    ip_hash TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analytics_events_event_occurred_at
    ON analytics_events (event_name, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_events_visitor_occurred_at
    ON analytics_events (visitor_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_events_job_occurred_at
    ON analytics_events (job_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_events_metadata_gin
    ON analytics_events USING GIN (metadata);
