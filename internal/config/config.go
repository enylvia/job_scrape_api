package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Database DatabaseConfig
}

type AppConfig struct {
	Name         string
	Environment  string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type AuthConfig struct {
	AdminUsername string
	AdminPassword string
	TokenSecret   string
	TokenTTL      time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int
}

type DatabaseConfig struct {
	Enabled         bool
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	Driver          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{
		App: AppConfig{
			Name:         getEnv("APP_NAME", "job-aggregator"),
			Environment:  getEnv("APP_ENV", "development"),
			Port:         getEnv("APP_PORT", "8080"),
			ReadTimeout:  getEnvDuration("APP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getEnvDuration("APP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getEnvDuration("APP_IDLE_TIMEOUT", 30*time.Second),
		},
		Auth: AuthConfig{
			AdminUsername: getEnv("ADMIN_USERNAME", ""),
			AdminPassword: getEnv("ADMIN_PASSWORD", ""),
			TokenSecret:   getEnv("ADMIN_TOKEN_SECRET", ""),
			TokenTTL:      getEnvDuration("ADMIN_TOKEN_TTL", 12*time.Hour),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvCSV("CORS_ALLOWED_ORIGINS", defaultCORSOrigins(getEnv("APP_ENV", "development"))),
			AllowedMethods: getEnvCSV("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}),
			AllowedHeaders: getEnvCSV("CORS_ALLOWED_HEADERS", []string{"Authorization", "Content-Type"}),
			MaxAge:         getEnvInt("CORS_MAX_AGE", 600),
		},
		Database: DatabaseConfig{
			Enabled:         getEnvBool("DB_ENABLED", false),
			Host:            getEnv("DB_HOST", "127.0.0.1"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "job_aggregator"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			Driver:          getEnv("DB_DRIVER", "pgx"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
	}

	if cfg.App.Port == "" {
		return Config{}, fmt.Errorf("APP_PORT must not be empty")
	}
	if cfg.Auth.AdminUsername == "" {
		return Config{}, fmt.Errorf("ADMIN_USERNAME must not be empty")
	}
	if cfg.Auth.AdminPassword == "" {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD must not be empty")
	}
	if cfg.Auth.TokenSecret == "" {
		return Config{}, fmt.Errorf("ADMIN_TOKEN_SECRET must not be empty")
	}
	if cfg.Auth.TokenTTL <= 0 {
		return Config{}, fmt.Errorf("ADMIN_TOKEN_TTL must be positive")
	}
	if cfg.App.Environment == "production" && len(cfg.CORS.AllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must be set in production")
	}
	for _, origin := range cfg.CORS.AllowedOrigins {
		if cfg.App.Environment == "production" && origin == "*" {
			return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain * in production")
		}
	}

	return cfg, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %s from %s: %w", key, path, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", path, err)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvCSV(key string, fallback []string) []string {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}

	return values
}

func defaultCORSOrigins(environment string) []string {
	if environment == "production" {
		return nil
	}

	return []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
	}
}
