package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/service"
)

// ApiAvailabilityHandler returns open booking slots for a given date.
type ApiAvailabilityHandler struct {
	Queries *db.Queries
}

// Routes returns the availability API sub-router.
// Mount at /availability on the API router.
func (h *ApiAvailabilityHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	return r
}

// Get calculates available slots for ?date=YYYY-MM-DD.
func (h *ApiAvailabilityHandler) Get(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date query parameter required (YYYY-MM-DD)"})
		return
	}

	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	// day_of_week: 0=Sunday … 6=Saturday (matches time.Weekday)
	dayOfWeek := int64(parsed.Weekday())

	schedule, err := h.Queries.GetScheduleByDay(r.Context(), dayOfWeek)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	booked, err := h.Queries.ListBookingsByDate(r.Context(), dateStr)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	slots := service.Calculate(dateStr, schedule, booked)
	if slots == nil {
		slots = []service.TimeSlot{}
	}
	writeJSON(w, http.StatusOK, slots)
}
