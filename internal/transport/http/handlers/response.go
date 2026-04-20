package handlers

import (
	"encoding/json"
	"net/http"
)

type apiResponse struct {
	APIMessage string `json:"api_message"`
	Count      int    `json:"count"`
	Data       any    `json:"data"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeData(w http.ResponseWriter, status int, apiMessage string, count int, data any) {
	writeJSON(w, status, apiResponse{
		APIMessage: apiMessage,
		Count:      count,
		Data:       data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiResponse{
		APIMessage: message,
		Count:      0,
		Data:       nil,
	})
}
