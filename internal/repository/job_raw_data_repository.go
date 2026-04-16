package repository

import (
	"context"
	"database/sql"
	"fmt"

	"job_aggregator/internal/models"
)

type JobRawDataRepository struct {
	db *sql.DB
}

func NewJobRawDataRepository(db *sql.DB) *JobRawDataRepository {
	return &JobRawDataRepository{db: db}
}

func (r *JobRawDataRepository) Create(ctx context.Context, data models.JobRawData) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database is not configured")
	}

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO job_raw_data (job_id, raw_html, raw_json, scraped_at)
		VALUES ($1, $2, NULLIF($3, '')::jsonb, $4)
		RETURNING id
	`,
		data.JobID,
		data.RawHTML,
		data.RawJSON,
		data.ScrapedAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert job raw data: %w", err)
	}

	return id, nil
}
