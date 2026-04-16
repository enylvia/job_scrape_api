CREATE TABLE IF NOT EXISTS jobs (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES sources(id),
    source_job_url TEXT NOT NULL,
    source_apply_url TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    slug TEXT NOT NULL DEFAULT '',
    company TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    employment_type TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    salary_min BIGINT NULL,
    salary_max BIGINT NULL,
    currency TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    requirements TEXT NOT NULL DEFAULT '',
    benefits TEXT NOT NULL DEFAULT '',
    posted_at TIMESTAMPTZ NULL,
    expired_at TIMESTAMPTZ NULL,
    content_hash TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    duplicate_of_job_id BIGINT NULL REFERENCES jobs(id),
    wordpress_post_id BIGINT NULL,
    telegram_sent BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_source_id ON jobs(source_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_content_hash ON jobs(content_hash);

