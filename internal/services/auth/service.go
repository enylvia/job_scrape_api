package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"job_aggregator/internal/config"
	"job_aggregator/internal/models"
)

type contextKey string

const claimsContextKey contextKey = "admin_auth_claims"

type Service struct {
	adminUserStore AdminUserStore
	tokenSecret    []byte
	tokenTTL       time.Duration
}

type AdminUserStore interface {
	GetActiveByUsername(ctx context.Context, username string) (models.AdminUser, error)
	MarkLogin(ctx context.Context, adminUserID int64) error
}

type Claims struct {
	Subject   string `json:"sub"`
	Username  string `json:"username"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	Username  string
}

func NewService(cfg config.AuthConfig, adminUserStore AdminUserStore) *Service {
	return &Service{
		adminUserStore: adminUserStore,
		tokenSecret:    []byte(cfg.TokenSecret),
		tokenTTL:       cfg.TokenTTL,
	}
}

func (s *Service) Login(ctx context.Context, username, password string, now time.Time) (LoginResult, error) {
	if s.adminUserStore == nil {
		return LoginResult{}, fmt.Errorf("admin user store is not configured")
	}

	adminUser, err := s.adminUserStore.GetActiveByUsername(ctx, username)
	if err != nil {
		return LoginResult{}, fmt.Errorf("invalid username or password")
	}
	if !CheckPasswordHash(password, adminUser.PasswordHash) {
		return LoginResult{}, fmt.Errorf("invalid username or password")
	}

	expiresAt := now.Add(s.tokenTTL)
	claims := Claims{
		Subject:   "admin",
		Username:  adminUser.Username,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}

	token, err := s.Sign(claims)
	if err != nil {
		return LoginResult{}, err
	}

	if err := s.adminUserStore.MarkLogin(ctx, adminUser.ID); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Token:     token,
		ExpiresAt: expiresAt,
		Username:  adminUser.Username,
	}, nil
}

func (s *Service) ValidateToken(token string, now time.Time) (Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Claims{}, fmt.Errorf("token is required")
	}

	payloadPart, signaturePart, ok := strings.Cut(token, ".")
	if !ok {
		return Claims{}, fmt.Errorf("invalid token")
	}
	if !s.signatureMatches(payloadPart, signaturePart) {
		return Claims{}, fmt.Errorf("invalid token")
	}

	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return Claims{}, fmt.Errorf("decode token payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, fmt.Errorf("parse token payload: %w", err)
	}
	if claims.Subject != "admin" || claims.Username == "" {
		return Claims{}, fmt.Errorf("invalid token claims")
	}
	if now.Unix() >= claims.ExpiresAt {
		return Claims{}, fmt.Errorf("token expired")
	}

	return claims, nil
}

func (s *Service) Sign(claims Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal token payload: %w", err)
	}

	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signaturePart := s.signPayload(payloadPart)
	return payloadPart + "." + signaturePart, nil
}

func (s *Service) ExtractBearerToken(authorizationHeader string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(authorizationHeader, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(authorizationHeader, prefix))
}

func (s *Service) WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(Claims)
	return claims, ok
}

func (s *Service) signatureMatches(payloadPart, signaturePart string) bool {
	expectedSignature := s.signPayload(payloadPart)
	return subtle.ConstantTimeCompare([]byte(signaturePart), []byte(expectedSignature)) == 1
}

func (s *Service) signPayload(payloadPart string) string {
	mac := hmac.New(sha256.New, s.tokenSecret)
	_, _ = mac.Write([]byte(payloadPart))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(hash), nil
}

func CheckPasswordHash(password, passwordHash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}
