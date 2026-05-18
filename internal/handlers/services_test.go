package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/solaris-soft/heartcave-backend/internal/testutils"
)

func setupServicesTest(t *testing.T) (*sql.DB, database.Querier, func()) {
	db, cleanup := testutils.SetupTestDB(t)
	queries := database.New(db)
	return db, queries, cleanup
}

func TestCreateService(t *testing.T) {
	_, queries, cleanup := setupServicesTest(t)
	defer cleanup()

	handler := NewServicesHandler(queries)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "creates service successfully",
			body:       `{"name":"Test Service","cents":5000,"description":"A test service","session_minutes":60}`,
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "fails with zero session minutes",
			body:       `{"name":"Bad Service","cents":1000,"description":"Bad","session_minutes":0}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with negative session minutes",
			body:       `{"name":"Bad Service","cents":1000,"description":"Bad","session_minutes":-10}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid JSON",
			body:       `{"name":`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with duplicate name",
			body:       `{"name":"Test Service","cents":3000,"description":"Duplicate","session_minutes":30}`,
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateService(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr {
				var resp database.Service
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID == uuid.Nil {
					t.Error("expected non-nil service ID")
				}
				if resp.Name != "Test Service" {
					t.Errorf("name = %q, want %q", resp.Name, "Test Service")
				}
				if resp.Price != "50.00" {
					t.Errorf("price = %q, want %q", resp.Price, "50.00")
				}
			}
		})
	}
}

func TestGetServices(t *testing.T) {
	_, queries, cleanup := setupServicesTest(t)
	defer cleanup()

	handler := NewServicesHandler(queries)

	// Create some services
	ctx := context.Background()
	_, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Service 1",
		Price:          "10.00",
		Description:    "First service",
		SessionMinutes: 30,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	_, err = queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Service 2",
		Price:          "20.00",
		Description:    "Second service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()

	handler.GetServices(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var services []database.Service
	if err := json.Unmarshal(rec.Body.Bytes(), &services); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(services) != 2 {
		t.Errorf("len(services) = %d, want 2", len(services))
	}
}

func TestGetService(t *testing.T) {
	_, queries, cleanup := setupServicesTest(t)
	defer cleanup()

	handler := NewServicesHandler(queries)

	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Get Test",
		Price:          "15.00",
		Description:    "Get test service",
		SessionMinutes: 45,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name       string
		pathID     string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "gets existing service",
			pathID:     service.ID.String(),
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "returns 404 for missing service",
			pathID:     uuid.New().String(),
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "returns 400 for invalid UUID",
			pathID:     "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "returns 400 for empty ID",
			pathID:     "",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/services/"+tt.pathID, nil)
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Get("/services/{id}", handler.GetService)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr {
				var resp database.Service
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != service.ID {
					t.Errorf("ID = %v, want %v", resp.ID, service.ID)
				}
			}
		})
	}
}

func TestUpdateService(t *testing.T) {
	_, queries, cleanup := setupServicesTest(t)
	defer cleanup()

	handler := NewServicesHandler(queries)

	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Update Test",
		Price:          "25.00",
		Description:    "Update test service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name       string
		pathID     string
		body       string
		wantStatus int
		wantErr    bool
		checkName  string
	}{
		{
			name:       "updates name",
			pathID:     service.ID.String(),
			body:       `{"name":"Updated Name"}`,
			wantStatus: http.StatusOK,
			wantErr:    false,
			checkName:  "Updated Name",
		},
		{
			name:       "updates cents",
			pathID:     service.ID.String(),
			body:       `{"cents":7500}`,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "updates description",
			pathID:     service.ID.String(),
			body:       `{"description":"New description"}`,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "updates session minutes",
			pathID:     service.ID.String(),
			body:       `{"session_minutes":90}`,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "fails with empty body",
			pathID:     service.ID.String(),
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid UUID",
			pathID:     "not-a-uuid",
			body:       `{"name":"Bad"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "returns 404 for missing service",
			pathID:     uuid.New().String(),
			body:       `{"name":"Missing"}`,
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/services/"+tt.pathID, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Patch("/services/{id}", handler.UpdateService)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr && tt.checkName != "" {
				var resp database.Service
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Name != tt.checkName {
					t.Errorf("name = %q, want %q", resp.Name, tt.checkName)
				}
			}
		})
	}
}

func TestDeleteService(t *testing.T) {
	_, queries, cleanup := setupServicesTest(t)
	defer cleanup()

	handler := NewServicesHandler(queries)

	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Delete Test",
		Price:          "5.00",
		Description:    "Delete test service",
		SessionMinutes: 15,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name       string
		pathID     string
		wantStatus int
	}{
		{
			name:       "deletes existing service",
			pathID:     service.ID.String(),
			wantStatus: http.StatusOK,
		},
		{
			name:       "returns 400 for invalid UUID",
			pathID:     "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns 400 for empty ID",
			pathID:     "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/services/"+tt.pathID, nil)
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Delete("/services/{id}", handler.DeleteService)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
