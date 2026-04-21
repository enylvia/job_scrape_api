package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"job_aggregator/internal/config"
	"job_aggregator/internal/services/auth"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func loggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		startedAt := time.Now()
		next.ServeHTTP(recorder, r)
		logger.Printf(
			"http request: method=%s path=%s status=%d duration=%s remote=%s query=%s",
			r.Method,
			r.URL.Path,
			recorder.statusCode,
			time.Since(startedAt).Round(time.Millisecond),
			r.RemoteAddr,
			r.URL.Query(),
		)
	})
}

func corsMiddleware(cfg config.CORSConfig, next http.Handler) http.Handler {
	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		allowedOrigins[strings.TrimSpace(origin)] = struct{}{}
	}

	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !isAllowedOrigin(origin, allowedOrigins) {
			if r.Method == http.MethodOptions {
				writeAuthError(w, http.StatusForbidden, "origin is not allowed")
				return
			}

			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		w.Header().Set("Access-Control-Max-Age", maxAge)
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authMiddleware(authService *auth.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresAdminAuth(r) {
			next.ServeHTTP(w, r)
			return
		}

		token := authService.ExtractBearerToken(r.Header.Get("Authorization"))
		claims, err := authService.ValidateToken(token, time.Now())
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		next.ServeHTTP(w, r.WithContext(authService.WithClaims(r.Context(), claims)))
	})
}

func isAllowedOrigin(origin string, allowedOrigins map[string]struct{}) bool {
	if _, ok := allowedOrigins["*"]; ok {
		return true
	}

	_, ok := allowedOrigins[origin]
	return ok
}

func requiresAdminAuth(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/internal/") {
		return false
	}
	if isPublicInternalEndpoint(r) {
		return false
	}

	return r.URL.Path != "/internal/auth/login"
}

func isPublicInternalEndpoint(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	if r.URL.Path == "/internal/about" {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/internal/about/") {
		return true
	}
	if r.URL.Path == "/internal/jobs" || r.URL.Path == "/internal/jobs/categories" {
		return true
	}

	return isJobDetailPath(r.URL.Path)
}

func isJobDetailPath(path string) bool {
	const prefix = "/internal/jobs/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	jobID := strings.TrimPrefix(path, prefix)
	if jobID == "" {
		return false
	}

	for _, r := range jobID {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(struct {
		APIMessage string `json:"api_message"`
		Count      int    `json:"count"`
		Data       any    `json:"data"`
	}{
		APIMessage: message,
		Count:      0,
		Data:       nil,
	})
}
