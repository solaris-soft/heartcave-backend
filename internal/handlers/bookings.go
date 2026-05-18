package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/services"
)

type BookingsHandler struct {
	BookingService services.BookingService
}

type CreateBookingRequest struct {
	ServiceID     string `json:"service_id"`
	StartsAt      string `json:"starts_at"`
	CustomerNotes string `json:"customer_notes"`
	SuccessURL    string `json:"success_url"`
	CancelURL     string `json:"cancel_url"`
}

type CreateBookingResponse struct {
	ID           uuid.UUID `json:"id"`
	Status       string    `json:"status"`
	ServiceName  string    `json:"service_name"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
	CheckoutURL  string    `json:"checkout_url"`
}

func (h BookingsHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req CreateBookingRequest
	if err := DecodeJson(r, &req); err != nil {
		WriteBadRequest(w)
		return
	}

	serviceID, err := uuid.Parse(req.ServiceID)
	if err != nil {
		WriteBadRequest(w)
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		WriteBadRequest(w)
		return
	}

	if req.SuccessURL == "" || req.CancelURL == "" {
		WriteBadRequest(w)
		return
	}

	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		WriteUnauthorized(w)
		return
	}

	result, err := h.BookingService.CreateBooking(
		r.Context(),
		userID,
		serviceID,
		startsAt,
		req.CustomerNotes,
		req.SuccessURL,
		req.CancelURL,
	)
	if err != nil {
		if errors.Is(err, services.ErrTimeslotUnavailable) {
			WriteJSON(w, http.StatusConflict, map[string]string{
				"error": "The requested timeslot is unavailable",
			})
			return
		}
		WriteInternalError(w)
		return
	}

	WriteJSON(w, http.StatusCreated, CreateBookingResponse{
		ID:          result.Booking.ID,
		Status:      result.Booking.Status,
		ServiceName: result.Booking.ServiceName,
		StartsAt:    result.Booking.StartsAt,
		EndsAt:      result.Booking.EndsAt,
		CheckoutURL: result.CheckoutURL,
	})
}
