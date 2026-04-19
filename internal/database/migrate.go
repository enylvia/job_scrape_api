package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ApplyMigrations(ctx context.Context, db *sql.DB, dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("glob migration files: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no migration files found in %s", dir)
	}

	sort.Strings(files)

	applied := make([]string, 0, len(files))
	for _, file := range files {
		if err := applySQLFile(ctx, db, file); err != nil {
			return applied, err
		}
		applied = append(applied, filepath.Base(file))
	}

	return applied, nil
}

func applySQLFile(ctx context.Context, db *sql.DB, path string) error {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("read migration %s: %w", path, err)
	}

	query := strings.TrimSpace(string(content))
	if query == "" {
		return nil
	}

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("execute migration %s: %w", path, err)
	}

	return nil
}
