package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func WriteBadRequest(w http.ResponseWriter) {
	WriteJSON(w, http.StatusBadRequest,
		map[string]string{"error": "Invalid request"})
}

func WriteInternalError(w http.ResponseWriter) {
	WriteJSON(w, http.StatusInternalServerError,
		map[string]string{"error": "Something went wrong"})
}

func WriteUnauthorized(w http.ResponseWriter) {
	WriteJSON(w, http.StatusUnauthorized,
		map[string]string{"error": "Not authorized"})
}

func WriteForbidden(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden,
		map[string]string{"error": "Forbidden"})
}

func DecodeJson(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}
