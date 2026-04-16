package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"job_aggregator/internal/models"
)

type SourceRepository struct {
	db *sql.DB
}

func NewSourceRepository(db *sql.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

func (r *SourceRepository) List(ctx context.Context) ([]models.Source, error) {
	if r.db == nil {
		return []models.Source{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, base_url, mode, active, scrape_interval_minutes, last_scraped_at, created_at, updated_at
		FROM sources
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query sources: %w", err)
	}
	defer rows.Close()

	sources := make([]models.Source, 0)
	for rows.Next() {
		var source models.Source
		if err := rows.Scan(
			&source.ID,
			&source.Name,
			&source.BaseURL,
			&source.Mode,
			&source.Active,
			&source.ScrapeIntervalMinutes,
			&source.LastScrapedAt,
			&source.CreatedAt,
			&source.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}

		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sources: %w", err)
	}

	return sources, nil
}

func (r *SourceRepository) ListActive(ctx context.Context) ([]models.Source, error) {
	if r.db == nil {
		return []models.Source{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, base_url, mode, active, scrape_interval_minutes, last_scraped_at, created_at, updated_at
		FROM sources
		WHERE active = TRUE
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query active sources: %w", err)
	}
	defer rows.Close()

	sources := make([]models.Source, 0)
	for rows.Next() {
		var source models.Source
		if err := rows.Scan(
			&source.ID,
			&source.Name,
			&source.BaseURL,
			&source.Mode,
			&source.Active,
			&source.ScrapeIntervalMinutes,
			&source.LastScrapedAt,
			&source.CreatedAt,
			&source.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan active source: %w", err)
		}

		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active sources: %w", err)
	}

	return sources, nil
}

func (r *SourceRepository) Create(ctx context.Context, source models.Source) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database is not configured")
	}

	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sources (name, base_url, mode, active, scrape_interval_minutes, last_scraped_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`,
		source.Name,
		source.BaseURL,
		source.Mode,
		source.Active,
		source.ScrapeIntervalMinutes,
		source.LastScrapedAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert source: %w", err)
	}

	return id, nil
}

func (r *SourceRepository) MarkScraped(ctx context.Context, sourceID int64, scrapedAt time.Time) error {
	if r.db == nil {
		return fmt.Errorf("database is not configured")
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE sources
		SET last_scraped_at = $2, updated_at = NOW()
		WHERE id = $1
	`, sourceID, scrapedAt)
	if err != nil {
		return fmt.Errorf("update source last_scraped_at: %w", err)
	}

	return nil
}
