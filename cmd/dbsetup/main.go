package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"job_aggregator/internal/config"
	"job_aggregator/internal/database"
)

type sourceSeed struct {
	Name                  string
	BaseURL               string
	Mode                  string
	ScrapeIntervalMinutes int
}

func main() {
	applyMigrations := flag.Bool("migrate", true, "apply SQL migrations")
	seedDefaultSources := flag.Bool("seed", true, "seed default sources")
	flag.Parse()

	if !*applyMigrations && !*seedDefaultSources {
		log.Fatalf("nothing to do: enable -migrate and/or -seed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	cfg.Database.Enabled = true

	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() {
		_ = database.Close(db)
	}()

	if *applyMigrations {
		applied, err := database.ApplyMigrations(ctx, db, "migrations")
		if err != nil {
			log.Fatalf("apply migrations: %v", err)
		}

		for _, file := range applied {
			log.Printf("applied migration: %s", file)
		}
	}

	if *seedDefaultSources {
		if err := seedSources(ctx, db, defaultSources()); err != nil {
			log.Fatalf("seed sources: %v", err)
		}
	}

	log.Printf("database setup completed successfully (migrate=%t seed=%t)", *applyMigrations, *seedDefaultSources)
}

func defaultSources() []sourceSeed {
	return []sourceSeed{
		{
			Name:                  "dealls",
			BaseURL:               "https://dealls.com/",
			Mode:                  "http",
			ScrapeIntervalMinutes: 60,
		},
		{
			Name:                  "glints",
			BaseURL:               "https://glints.com/",
			Mode:                  "browser",
			ScrapeIntervalMinutes: 60,
		},
	}
}

func seedSources(ctx context.Context, db *sql.DB, seeds []sourceSeed) error {
	for _, seed := range seeds {
		if err := upsertSource(ctx, db, seed); err != nil {
			return err
		}
	}

	return nil
}

func upsertSource(ctx context.Context, db *sql.DB, seed sourceSeed) error {
	result, err := db.ExecContext(ctx, `
		UPDATE sources
		SET
			base_url = $2,
			mode = $3,
			active = TRUE,
			scrape_interval_minutes = $4,
			updated_at = NOW()
		WHERE name = $1
	`, seed.Name, seed.BaseURL, seed.Mode, seed.ScrapeIntervalMinutes)
	if err != nil {
		return fmt.Errorf("update source %s: %w", seed.Name, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for source %s: %w", seed.Name, err)
	}

	if rowsAffected > 0 {
		log.Printf("updated source: %s", seed.Name)
		return nil
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO sources (
			name,
			base_url,
			mode,
			active,
			scrape_interval_minutes
		) VALUES ($1, $2, $3, TRUE, $4)
	`, seed.Name, seed.BaseURL, seed.Mode, seed.ScrapeIntervalMinutes)
	if err != nil {
		return fmt.Errorf("insert source %s: %w", seed.Name, err)
	}

	log.Printf("inserted source: %s", seed.Name)
	return nil
}
