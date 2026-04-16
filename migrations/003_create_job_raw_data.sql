CREATE TABLE IF NOT EXISTS job_raw_data (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    raw_html TEXT NOT NULL DEFAULT '',
    raw_json JSONB NULL,
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_job_raw_data_job_id ON job_raw_data(job_id);

