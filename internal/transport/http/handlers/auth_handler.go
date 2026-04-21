package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"job_aggregator/internal/services/auth"
)

type AuthHandler struct {
	logger      *log.Logger
	authService *auth.Service
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
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authService.Login(r.Context(), request.Username, request.Password, time.Now())
	if err != nil {
		h.logger.Printf("auth handler: login failed username=%q error=%v", request.Username, err)
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	writeData(w, http.StatusOK, "login successful", 1, loginResponse{
		TokenType: "Bearer",
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		Admin: adminInfo{
			Username: result.Username,
		},
	})
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
