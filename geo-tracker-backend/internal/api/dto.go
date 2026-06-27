package api

import (
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, message string, code string) {
	sendJSON(w, status, ErrorResponse{Error: message, Code: code})
}
