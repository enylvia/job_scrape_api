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

func (r *JobRepository) ListByStatus(ctx context.Context, status string) ([]models.Job, error) {
	if r.db == nil {
		return []models.Job{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, source_id, source_job_url, source_apply_url, title, slug, company, location,
			employment_type, category, salary_min, salary_max, currency, description, requirements,
			benefits, posted_at, expired_at, content_hash, status, duplicate_of_job_id,
			wordpress_post_id, telegram_sent, created_at, updated_at
		FROM jobs
		WHERE status = $1
		ORDER BY id ASC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("query jobs by status: %w", err)
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)
	for rows.Next() {
		var job models.Job
		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("scan job by status: %w", err)
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs by status: %w", err)
	}

	return jobs, nil
}

func (r *JobRepository) UpdateNormalized(ctx context.Context, job models.Job) error {
	if r.db == nil {
		return fmt.Errorf("database is not configured")
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE jobs
		SET
			source_job_url = $2,
			source_apply_url = $3,
			title = $4,
			slug = $5,
			company = $6,
			location = $7,
			employment_type = $8,
			category = $9,
			description = $10,
			requirements = $11,
			benefits = $12,
			content_hash = $13,
			status = $14,
			updated_at = NOW()
		WHERE id = $1
	`,
		job.ID,
		job.SourceJobURL,
		job.SourceApplyURL,
		job.Title,
		job.Slug,
		job.Company,
		job.Location,
		job.EmploymentType,
		job.Category,
		job.Description,
		job.Requirements,
		job.Benefits,
		job.ContentHash,
		job.Status,
	)
	if err != nil {
		return fmt.Errorf("update normalized job: %w", err)
	}

	return nil
}

func (r *JobRepository) MarkDuplicate(ctx context.Context, jobID, duplicateOfJobID int64) error {
	if r.db == nil {
		return fmt.Errorf("database is not configured")
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE jobs
		SET
			status = 'duplicate',
			duplicate_of_job_id = $2,
			updated_at = NOW()
		WHERE id = $1
	`, jobID, duplicateOfJobID)
	if err != nil {
		return fmt.Errorf("mark duplicate job: %w", err)
	}

	return nil
}
