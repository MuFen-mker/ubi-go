package api

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func WriteSuccess(w http.ResponseWriter, statusCode int, data interface{}, message string) {
	resp := Response{
		Success: true,
		Message: message,
		Data:    data,
	}
	WriteJSON(w, statusCode, resp)
}

func WriteError(w http.ResponseWriter, statusCode int, err string) {
	resp := Response{
		Success: false,
		Error:   err,
	}
	WriteJSON(w, statusCode, resp)
}
