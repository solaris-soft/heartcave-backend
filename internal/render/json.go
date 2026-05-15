package render

import (
	"encoding/json"
	"net/http"
)

type JSON map[string]any

// WriteJson writes a json response
func WriteJson(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(data)
}

// WriteError writes a json error response
func WriteError(w http.ResponseWriter, status int, message string) error {
	return WriteJson(w, status, JSON{
		"error": message,
	})
}

// WriteValidationErrors takes a map of errors and returns it as json
func WriteValidationErrors(w http.ResponseWriter, errors map[string]string) {
	_ = WriteJson(w, http.StatusBadRequest, JSON{
		"errors": errors,
	})
}

// DecodeJson decodes a JSON request body into dst.
func DecodeJson(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}
