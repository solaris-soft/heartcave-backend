package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantStatus int
		wantBody   string
		wantCT     string
	}{
		{
			name:       "writes JSON response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "hello"},
			wantStatus: http.StatusOK,
			wantBody:   `{"message":"hello"}` + "\n",
			wantCT:     "application/json",
		},
		{
			name:       "writes empty object",
			status:     http.StatusCreated,
			data:       map[string]string{},
			wantStatus: http.StatusCreated,
			wantBody:   "{}\n",
			wantCT:     "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteJSON(rec, tt.status, tt.data)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
			ct := rec.Header().Get("Content-Type")
			if ct != tt.wantCT {
				t.Errorf("content-type = %q, want %q", ct, tt.wantCT)
			}
		})
	}
}

func TestErrorWriters(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(http.ResponseWriter)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "WriteBadRequest",
			fn:         WriteBadRequest,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"Invalid request"}` + "\n",
		},
		{
			name:       "WriteInternalError",
			fn:         WriteInternalError,
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Something went wrong"}` + "\n",
		},
		{
			name:       "WriteUnauthorized",
			fn:         WriteUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"error":"Not authorized"}` + "\n",
		},
		{
			name:       "WriteForbidden",
			fn:         WriteForbidden,
			wantStatus: http.StatusForbidden,
			wantBody:   `{"error":"Forbidden"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.fn(rec)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestDecodeJson(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
		want    map[string]string
	}{
		{
			name:    "decodes valid JSON",
			body:    `{"key":"value"}`,
			wantErr: false,
			want:    map[string]string{"key": "value"},
		},
		{
			name:    "errors on invalid JSON",
			body:    `{"key":`,
			wantErr: true,
		},
		{
			name:    "decodes empty object",
			body:    `{}`,
			wantErr: false,
			want:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			var dst map[string]string
			err := DecodeJson(req, &dst)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !jsonEqual(t, dst, tt.want) {
				t.Errorf("got %v, want %v", dst, tt.want)
			}
		})
	}
}

func jsonEqual(t *testing.T, a, b any) bool {
	t.Helper()
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
