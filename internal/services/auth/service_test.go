package auth

import (
	"context"
	"testing"
	"time"

	"job_aggregator/internal/config"
	"job_aggregator/internal/models"
)

func TestServiceLoginAndValidateToken(t *testing.T) {
	store := newTestAdminUserStore(t, "admin", "secret")
	service := NewService(config.AuthConfig{
		TokenSecret: "test-secret",
		TokenTTL:    time.Hour,
	}, store)

	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	result, err := service.Login(context.Background(), "admin", "secret", now)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	claims, err := service.ValidateToken(result.Token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.Username != "admin" {
		t.Fatalf("claims.Username = %q, want %q", claims.Username, "admin")
	}
}

func TestServiceRejectsExpiredToken(t *testing.T) {
	store := newTestAdminUserStore(t, "admin", "secret")
	service := NewService(config.AuthConfig{
		TokenSecret: "test-secret",
		TokenTTL:    time.Hour,
	}, store)

	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	result, err := service.Login(context.Background(), "admin", "secret", now)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if _, err := service.ValidateToken(result.Token, now.Add(2*time.Hour)); err == nil {
		t.Fatal("ValidateToken() error = nil, want expired token error")
	}
}

func TestServiceRejectsInvalidCredentials(t *testing.T) {
	store := newTestAdminUserStore(t, "admin", "secret")
	service := NewService(config.AuthConfig{
		TokenSecret: "test-secret",
		TokenTTL:    time.Hour,
	}, store)

	if _, err := service.Login(context.Background(), "admin", "wrong", time.Now()); err == nil {
		t.Fatal("Login() error = nil, want invalid credentials error")
	}
}

type testAdminUserStore struct {
	user models.AdminUser
}

func newTestAdminUserStore(t *testing.T, username, password string) *testAdminUserStore {
	t.Helper()

	passwordHash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	return &testAdminUserStore{
		user: models.AdminUser{
			ID:           1,
			Username:     username,
			PasswordHash: passwordHash,
			IsActive:     true,
		},
	}
}

func (s *testAdminUserStore) GetActiveByUsername(_ context.Context, username string) (models.AdminUser, error) {
	if username != s.user.Username {
		return models.AdminUser{}, context.Canceled
	}

	return s.user, nil
}

func (s *testAdminUserStore) MarkLogin(_ context.Context, _ int64) error {
	return nil
}
