package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/solaris-soft/heartcave-backend/internal/services"
)

// mockBookingService implements services.BookingService for testing
type mockBookingService struct {
	result services.CreateBookingResult
	err    error
}

func (m *mockBookingService) CreateBooking(
	ctx context.Context,
	customerID uuid.UUID,
	serviceID uuid.UUID,
	startsAt time.Time,
	customerNotes string,
	successURL string,
	cancelURL string,
) (services.CreateBookingResult, error) {
	return m.result, m.err
}

func TestCreateBooking(t *testing.T) {
	_, queries, authService, cleanup := setupAuthTest(t)
	defer cleanup()

	// Create a user
	ctx := context.Background()
	hash, _ := authService.HashPassword("password123")
	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		Name:         "Booking Test",
		Email:        "booking@example.com",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	accessToken, _ := authService.MakeAccessToken(user.ID, string(database.UserRoleCustomer))
	authHandler := NewAuthHandler(queries, authService)

	tests := []struct {
		name       string
		body       string
		mockResult services.CreateBookingResult
		mockErr    error
		wantStatus int
		wantErr    bool
		checkID    bool
	}{
		{
			name: "creates booking successfully",
			body: `{"service_id":"` + uuid.New().String() + `","starts_at":"2024-06-15T10:00:00Z","customer_notes":"Please bring water","success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}`,
			mockResult: services.CreateBookingResult{
				Booking: database.Booking{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Status:      "pending",
					ServiceName: "Test Service",
					StartsAt:    time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
					EndsAt:      time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC),
				},
				CheckoutURL: "https://checkout.stripe.com/test",
			},
			mockErr:    nil,
			wantStatus: http.StatusCreated,
			wantErr:    false,
			checkID:    true,
		},
		{
			name:       "returns conflict for unavailable timeslot",
			body:       `{"service_id":"` + uuid.New().String() + `","starts_at":"2024-06-15T10:00:00Z","success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}`,
			mockResult: services.CreateBookingResult{},
			mockErr:    services.ErrTimeslotUnavailable,
			wantStatus: http.StatusConflict,
			wantErr:    true,
		},
		{
			name:       "returns internal error for service error",
			body:       `{"service_id":"` + uuid.New().String() + `","starts_at":"2024-06-15T10:00:00Z","success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}`,
			mockResult: services.CreateBookingResult{},
			mockErr:    errors.New("some internal error"),
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "fails with invalid JSON",
			body:       `{"service_id":`,
			mockResult: services.CreateBookingResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid service ID",
			body:       `{"service_id":"not-a-uuid","starts_at":"2024-06-15T10:00:00Z","success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}`,
			mockResult: services.CreateBookingResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid time format",
			body:       `{"service_id":"` + uuid.New().String() + `","starts_at":"not-a-time","success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}`,
			mockResult: services.CreateBookingResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with missing success URL",
			body:       `{"service_id":"` + uuid.New().String() + `","starts_at":"2024-06-15T10:00:00Z","cancel_url":"https://example.com/cancel"}`,
			mockResult: services.CreateBookingResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with missing cancel URL",
			body:       `{"service_id":"` + uuid.New().String() + `","starts_at":"2024-06-15T10:00:00Z","success_url":"https://example.com/success"}`,
			mockResult: services.CreateBookingResult{},
			mockErr:    nil,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails without auth",
			body:       `{"service_id":"` + uuid.New().String() + `","starts_at":"2024-06-15T10:00:00Z","success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}`,
			mockResult: services.CreateBookingResult{},
			mockErr:    nil,
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockBookingService{
				result: tt.mockResult,
				err:    tt.mockErr,
			}
			bookingHandler := BookingsHandler{BookingService: mockSvc}

			req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.name != "fails without auth" {
				req.Header.Set("Authorization", "Bearer "+accessToken)
			}
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.With(authHandler.AuthMiddleware).Post("/bookings", bookingHandler.CreateBooking)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr && tt.checkID {
				var resp CreateBookingResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID == uuid.Nil {
					t.Error("expected non-nil booking ID")
				}
				if resp.CheckoutURL != tt.mockResult.CheckoutURL {
					t.Errorf("checkout URL = %q, want %q", resp.CheckoutURL, tt.mockResult.CheckoutURL)
				}
			}
		})
	}
}

func TestCreateBooking_Integration(t *testing.T) {
	_, queries, authService, cleanup := setupAuthTest(t)
	defer cleanup()

	// Create a user
	ctx := context.Background()
	hash, _ := authService.HashPassword("password123")
	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		Name:         "Booking Integration Test",
		Email:        "booking-int@example.com",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a service
	service, err := queries.CreateService(ctx, database.CreateServiceParams{
		Name:           "Integration Service",
		Price:          "50.00",
		Description:    "Test service",
		SessionMinutes: 60,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	accessToken, _ := authService.MakeAccessToken(user.ID, string(database.UserRoleCustomer))
	authHandler := NewAuthHandler(queries, authService)

	// This test uses a mock booking service to avoid Stripe calls
	mockSvc := &mockBookingService{
		result: services.CreateBookingResult{
			Booking: database.Booking{
				ID:          uuid.New(),
				Status:      "pending",
				ServiceName: service.Name,
				StartsAt:    time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
				EndsAt:      time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC),
			},
			CheckoutURL: "https://checkout.stripe.com/test",
		},
	}
	bookingHandler := BookingsHandler{BookingService: mockSvc}

	body := `{"service_id":"` + service.ID.String() + `","starts_at":"2024-06-15T10:00:00Z","customer_notes":"Integration test","success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}`

	req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	r.With(authHandler.AuthMiddleware).Post("/bookings", bookingHandler.CreateBooking)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp CreateBookingResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != "pending" {
		t.Errorf("status = %q, want %q", resp.Status, "pending")
	}
	if resp.ServiceName != service.Name {
		t.Errorf("service name = %q, want %q", resp.ServiceName, service.Name)
	}
}
