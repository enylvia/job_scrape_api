package repository

import (
	"context"
	"database/sql"
	"fmt"

	"job_aggregator/internal/models"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) GetByID(ctx context.Context, id int64) (models.Job, error) {
	if r.db == nil {
		return models.Job{}, fmt.Errorf("database is not configured")
	}

	var job models.Job
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id, source_id, source_job_url, source_apply_url, title, slug, company, location,
			employment_type, category, salary_min, salary_max, currency, description, requirements,
			benefits, posted_at, expired_at, content_hash, status, duplicate_of_job_id,
			wordpress_post_id, telegram_sent, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`, id).Scan(
		&job.ID,
		&job.SourceID,
		&job.SourceJobURL,
		&job.SourceApplyURL,
		&job.Title,
		&job.Slug,
		&job.Company,
		&job.Location,
		&job.EmploymentType,
		&job.Category,
		&job.SalaryMin,
		&job.SalaryMax,
		&job.Currency,
		&job.Description,
		&job.Requirements,
		&job.Benefits,
		&job.PostedAt,
		&job.ExpiredAt,
		&job.ContentHash,
		&job.Status,
		&job.DuplicateOfJobID,
		&job.WordPressPostID,
		&job.TelegramSent,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return models.Job{}, fmt.Errorf("get job by id: %w", err)
	}

	return job, nil
}

func (r *JobRepository) Create(ctx context.Context, job models.Job) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database is not configured")
	}

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO jobs (
			source_id, source_job_url, source_apply_url, title, slug, company, location,
			employment_type, category, salary_min, salary_max, currency, description, requirements,
			benefits, posted_at, expired_at, content_hash, status, duplicate_of_job_id,
			wordpress_post_id, telegram_sent
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20,
			$21, $22
		)
		RETURNING id
	`,
		job.SourceID,
		job.SourceJobURL,
		job.SourceApplyURL,
		job.Title,
		job.Slug,
		job.Company,
		job.Location,
		job.EmploymentType,
		job.Category,
		job.SalaryMin,
		job.SalaryMax,
		job.Currency,
		job.Description,
		job.Requirements,
		job.Benefits,
		job.PostedAt,
		job.ExpiredAt,
		job.ContentHash,
		job.Status,
		job.DuplicateOfJobID,
		job.WordPressPostID,
		job.TelegramSent,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert job: %w", err)
	}

	return id, nil
}

func (r *JobRepository) ExistsBySourceJobURL(ctx context.Context, sourceID int64, sourceJobURL string) (bool, error) {
	if r.db == nil {
		return false, fmt.Errorf("database is not configured")
	}

	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM jobs
			WHERE source_id = $1 AND source_job_url = $2
		)
	`, sourceID, sourceJobURL).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check job by source_job_url: %w", err)
	}

	return exists, nil
}
