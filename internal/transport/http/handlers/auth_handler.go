package handlers

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"job_aggregator/internal/services/auth"
)

const (
	maxLoginFailures = 5
	loginWindow      = 15 * time.Minute
	loginLockout     = 15 * time.Minute
)

type AuthHandler struct {
	logger      *log.Logger
	authService *auth.Service
	limiter     *loginRateLimiter
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	TokenType string    `json:"token_type"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Admin     adminInfo `json:"admin"`
}

type adminInfo struct {
	Username string `json:"username"`
}

func NewAuthHandler(logger *log.Logger, authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		logger:      logger,
		authService: authService,
		limiter:     newLoginRateLimiter(),
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var request loginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	limiterKey := loginLimiterKey(r, request.Username)
	if h.limiter.IsLocked(limiterKey, time.Now()) {
		writeError(w, http.StatusTooManyRequests, "too many login attempts, please try again later")
		return
	}

	result, err := h.authService.Login(r.Context(), request.Username, request.Password, time.Now())
	if err != nil {
		h.limiter.RecordFailure(limiterKey, time.Now())
		h.logger.Printf("auth handler: login failed username=%q error=%v", request.Username, err)
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	h.limiter.Reset(limiterKey)
	writeData(w, http.StatusOK, "login successful", 1, loginResponse{
		TokenType: "Bearer",
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		Admin: adminInfo{
			Username: result.Username,
		},
	})
}

type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

type loginAttempt struct {
	Failures      int
	WindowStarted time.Time
	LockedUntil   time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{
		attempts: make(map[string]loginAttempt),
	}
}

func (l *loginRateLimiter) IsLocked(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt := l.attempts[key]
	if attempt.LockedUntil.IsZero() {
		return false
	}
	if now.Before(attempt.LockedUntil) {
		return true
	}

	delete(l.attempts, key)
	return false
}

func (l *loginRateLimiter) RecordFailure(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt := l.attempts[key]
	if attempt.WindowStarted.IsZero() || now.Sub(attempt.WindowStarted) > loginWindow {
		attempt = loginAttempt{
			WindowStarted: now,
		}
	}

	attempt.Failures++
	if attempt.Failures >= maxLoginFailures {
		attempt.LockedUntil = now.Add(loginLockout)
	}
	l.attempts[key] = attempt
}

func (l *loginRateLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, key)
}

func loginLimiterKey(r *http.Request, username string) string {
	return clientIP(r) + "|" + strings.ToLower(strings.TrimSpace(username))
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	writeData(w, http.StatusOK, "admin profile fetched successfully", 1, adminInfo{
		Username: claims.Username,
	})
}
