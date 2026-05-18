package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
)

func setupAvailabilityTest(t *testing.T) (database.Querier, func()) {
	_, queries, cleanup := setupServicesTest(t)
	return queries, cleanup
}

func TestCreateAvailability(t *testing.T) {
	queries, cleanup := setupAvailabilityTest(t)
	defer cleanup()

	// Create a service first
	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Availability Test",
		Price:          "50.00",
		Description:    "Test service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	handler := NewServiceAvailabilityHandler(queries)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "creates availability successfully",
			body:       mustJSON(t, map[string]any{"service_id": service.ID.String(), "day_of_week": 1, "start_time": time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), "end_time": time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC)}),
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "fails with end before start",
			body:       mustJSON(t, map[string]any{"service_id": service.ID.String(), "day_of_week": 1, "start_time": time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC), "end_time": time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)}),
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid day of week",
			body:       mustJSON(t, map[string]any{"service_id": service.ID.String(), "day_of_week": 7, "start_time": time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), "end_time": time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC)}),
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with negative day of week",
			body:       mustJSON(t, map[string]any{"service_id": service.ID.String(), "day_of_week": -1, "start_time": time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), "end_time": time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC)}),
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid JSON",
			body:       `{"service_id":`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/availability", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateAvailability(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr {
				var resp database.ServiceAvailability
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID == uuid.Nil {
					t.Error("expected non-nil availability ID")
				}
			}
		})
	}
}

func TestGetAvailability(t *testing.T) {
	queries, cleanup := setupAvailabilityTest(t)
	defer cleanup()

	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Get Availability Test",
		Price:          "50.00",
		Description:    "Test service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	availability, err := queries.CreateAvailability(ctx, database.CreateAvailabilityParams{
		ServiceID: service.ID,
		DayOfWeek: 2,
		StartTime: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("failed to create availability: %v", err)
	}

	handler := NewServiceAvailabilityHandler(queries)

	tests := []struct {
		name       string
		pathID     string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "gets existing availability",
			pathID:     availability.ID.String(),
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "returns 404 for missing availability",
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
			req := httptest.NewRequest(http.MethodGet, "/availability/"+tt.pathID, nil)
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Get("/availability/{id}", handler.GetAvailability)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr {
				var resp database.ServiceAvailability
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != availability.ID {
					t.Errorf("ID = %v, want %v", resp.ID, availability.ID)
				}
			}
		})
	}
}

func TestGetAllAvailability(t *testing.T) {
	queries, cleanup := setupAvailabilityTest(t)
	defer cleanup()

	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "All Availability Test",
		Price:          "50.00",
		Description:    "Test service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = queries.CreateAvailability(ctx, database.CreateAvailabilityParams{
		ServiceID: service.ID,
		DayOfWeek: 1,
		StartTime: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("failed to create availability: %v", err)
	}

	_, err = queries.CreateAvailability(ctx, database.CreateAvailabilityParams{
		ServiceID: service.ID,
		DayOfWeek: 2,
		StartTime: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 1, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("failed to create availability: %v", err)
	}

	handler := NewServiceAvailabilityHandler(queries)

	req := httptest.NewRequest(http.MethodGet, "/availability", nil)
	rec := httptest.NewRecorder()

	handler.GetAllAvailability(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var availabilities []database.ServiceAvailability
	if err := json.Unmarshal(rec.Body.Bytes(), &availabilities); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(availabilities) != 2 {
		t.Errorf("len(availabilities) = %d, want 2", len(availabilities))
	}
}

func TestGetAvailabilityByService(t *testing.T) {
	queries, cleanup := setupAvailabilityTest(t)
	defer cleanup()

	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "By Service Test",
		Price:          "50.00",
		Description:    "Test service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = queries.CreateAvailability(ctx, database.CreateAvailabilityParams{
		ServiceID: service.ID,
		DayOfWeek: 1,
		StartTime: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("failed to create availability: %v", err)
	}

	handler := NewServiceAvailabilityHandler(queries)

	tests := []struct {
		name       string
		pathID     string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "gets availability by service",
			pathID:     service.ID.String(),
			wantStatus: http.StatusOK,
			wantErr:    false,
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
			req := httptest.NewRequest(http.MethodGet, "/services/"+tt.pathID+"/availability", nil)
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Get("/services/{service_id}/availability", handler.GetAvailabilityByService)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr {
				var resp []database.ServiceAvailability
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp) != 1 {
					t.Errorf("len(resp) = %d, want 1", len(resp))
				}
			}
		})
	}
}

func TestUpdateAvailability(t *testing.T) {
	queries, cleanup := setupAvailabilityTest(t)
	defer cleanup()

	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Update Availability Test",
		Price:          "50.00",
		Description:    "Test service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	availability, err := queries.CreateAvailability(ctx, database.CreateAvailabilityParams{
		ServiceID: service.ID,
		DayOfWeek: 1,
		StartTime: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("failed to create availability: %v", err)
	}

	handler := NewServiceAvailabilityHandler(queries)

	tests := []struct {
		name       string
		pathID     string
		body       string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "updates day of week",
			pathID:     availability.ID.String(),
			body:       `{"day_of_week":3}`,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "updates times",
			pathID:     availability.ID.String(),
			body:       mustJSON(t, map[string]any{"start_time": time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), "end_time": time.Date(2024, 1, 1, 16, 0, 0, 0, time.UTC)}),
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "fails with end before start",
			pathID:     availability.ID.String(),
			body:       mustJSON(t, map[string]any{"start_time": time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC), "end_time": time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)}),
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid day of week",
			pathID:     availability.ID.String(),
			body:       `{"day_of_week":7}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with empty body",
			pathID:     availability.ID.String(),
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "returns 404 for missing availability",
			pathID:     uuid.New().String(),
			body:       `{"day_of_week":2}`,
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "returns 400 for invalid UUID",
			pathID:     "not-a-uuid",
			body:       `{"day_of_week":2}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/availability/"+tt.pathID, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Patch("/availability/{id}", handler.UpdateAvailability)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestDeleteAvailability(t *testing.T) {
	queries, cleanup := setupAvailabilityTest(t)
	defer cleanup()

	ctx := context.Background()
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Delete Availability Test",
		Price:          "50.00",
		Description:    "Test service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	availability, err := queries.CreateAvailability(ctx, database.CreateAvailabilityParams{
		ServiceID: service.ID,
		DayOfWeek: 1,
		StartTime: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("failed to create availability: %v", err)
	}

	handler := NewServiceAvailabilityHandler(queries)

	tests := []struct {
		name       string
		pathID     string
		wantStatus int
	}{
		{
			name:       "deletes existing availability",
			pathID:     availability.ID.String(),
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
			req := httptest.NewRequest(http.MethodDelete, "/availability/"+tt.pathID, nil)
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Delete("/availability/{id}", handler.DeleteAvailability)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return string(b)
}
