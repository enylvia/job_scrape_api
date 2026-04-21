package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"job_aggregator/internal/models"
)

type AdminUserRepository struct {
	db *sql.DB
}

func NewAdminUserRepository(db *sql.DB) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

func (r *AdminUserRepository) GetActiveByUsername(ctx context.Context, username string) (models.AdminUser, error) {
	if r.db == nil {
		return models.AdminUser{}, fmt.Errorf("database is not configured")
	}

	var user models.AdminUser
	err := r.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, is_active, last_login_at, created_at, updated_at
		FROM admin_users
		WHERE LOWER(username) = LOWER($1) AND is_active = TRUE
	`, strings.TrimSpace(username)).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return models.AdminUser{}, fmt.Errorf("get active admin user by username: %w", err)
	}

	return user, nil
}

func (r *AdminUserRepository) MarkLogin(ctx context.Context, adminUserID int64) error {
	if r.db == nil {
		return fmt.Errorf("database is not configured")
	}

	if _, err := r.db.ExecContext(ctx, `
		UPDATE admin_users
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, adminUserID); err != nil {
		return fmt.Errorf("mark admin login: %w", err)
	}

	return nil
}

func (r *AdminUserRepository) EnsureBootstrapAdmin(ctx context.Context, username, password string) error {
	if r.db == nil {
		return nil
	}

	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil
	}

	var existingCount int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_users`).Scan(&existingCount); err != nil {
		return fmt.Errorf("count admin users: %w", err)
	}
	if existingCount > 0 {
		return nil
	}

	passwordHash, err := hashAdminPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO admin_users (username, password_hash)
		VALUES ($1, $2)
	`, username, passwordHash); err != nil {
		return fmt.Errorf("insert bootstrap admin user: %w", err)
	}

	return nil
}

func hashAdminPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(hash), nil
}
