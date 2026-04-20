package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"job_aggregator/internal/models"
)

type AboutPageRepository struct {
	db *sql.DB
}

func NewAboutPageRepository(db *sql.DB) *AboutPageRepository {
	return &AboutPageRepository{db: db}
}

func (r *AboutPageRepository) List(ctx context.Context) ([]models.AboutPage, int, error) {
	if r.db == nil {
		return []models.AboutPage{}, 0, nil
	}

	totalCount, err := r.countAll(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, body, created_at, updated_at
		FROM about_pages
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("query about pages: %w", err)
	}
	defer rows.Close()

	pages := make([]models.AboutPage, 0)
	for rows.Next() {
		var page models.AboutPage
		if err := rows.Scan(
			&page.ID,
			&page.Title,
			&page.Body,
			&page.CreatedAt,
			&page.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan about page: %w", err)
		}

		pages = append(pages, page)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate about pages: %w", err)
	}

	return pages, totalCount, nil
}

func (r *AboutPageRepository) GetByID(ctx context.Context, id int64) (models.AboutPage, error) {
	if r.db == nil {
		return models.AboutPage{}, fmt.Errorf("database is not configured")
	}

	var page models.AboutPage
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, body, created_at, updated_at
		FROM about_pages
		WHERE id = $1
	`, id).Scan(
		&page.ID,
		&page.Title,
		&page.Body,
		&page.CreatedAt,
		&page.UpdatedAt,
	)
	if err != nil {
		return models.AboutPage{}, fmt.Errorf("get about page by id: %w", err)
	}

	return page, nil
}

func (r *AboutPageRepository) Create(ctx context.Context, page models.AboutPage) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database is not configured")
	}

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO about_pages (title, body)
		VALUES ($1, $2)
		RETURNING id
	`,
		strings.TrimSpace(page.Title),
		normalizeStoredText(page.Body),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert about page: %w", err)
	}

	return id, nil
}

func (r *AboutPageRepository) Update(ctx context.Context, page models.AboutPage) error {
	if r.db == nil {
		return fmt.Errorf("database is not configured")
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE about_pages
		SET
			title = $2,
			body = $3,
			updated_at = NOW()
		WHERE id = $1
	`,
		page.ID,
		strings.TrimSpace(page.Title),
		normalizeStoredText(page.Body),
	)
	if err != nil {
		return fmt.Errorf("update about page: %w", err)
	}

	return nil
}

func (r *AboutPageRepository) Delete(ctx context.Context, id int64) error {
	if r.db == nil {
		return fmt.Errorf("database is not configured")
	}

	_, err := r.db.ExecContext(ctx, `DELETE FROM about_pages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete about page: %w", err)
	}

	return nil
}

func (r *AboutPageRepository) countAll(ctx context.Context) (int, error) {
	var totalCount int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM about_pages`).Scan(&totalCount); err != nil {
		return 0, fmt.Errorf("count about pages: %w", err)
	}

	return totalCount, nil
}
