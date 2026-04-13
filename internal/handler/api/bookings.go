package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/middleware"
)

// ApiBookingsHandler serves a customer's own booking history.
type ApiBookingsHandler struct {
	Queries *db.Queries
}

// Routes returns the customer bookings API sub-router.
// Mount at /bookings on the JWT-authenticated API router.
func (h *ApiBookingsHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/bookings", h.ListMine)
	return r
}

// ListMine returns all bookings for the authenticated customer.
func (h *ApiBookingsHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	rawID, ok := r.Context().Value(middleware.CustomerIDKey).(string)
	if !ok || rawID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	customerID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	bookings, err := h.Queries.ListBookingsByCustomer(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if bookings == nil {
		bookings = []db.ListBookingsByCustomerRow{}
	}
	writeJSON(w, http.StatusOK, bookings)
}
