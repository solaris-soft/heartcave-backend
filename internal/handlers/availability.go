package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/service"
)

type AvailabilityHandler struct {
	queries  *db.Queries
	logger   *slog.Logger
	location *time.Location
}

func NewAvailabilityHandler(queries *db.Queries, logger *slog.Logger, location *time.Location) AvailabilityHandler {
	return AvailabilityHandler{queries: queries, logger: logger, location: location}
}

func (h AvailabilityHandler) Get(w http.ResponseWriter, r *http.Request) {
	dateParam := r.URL.Query().Get("date")
	if dateParam == "" {
		writeError(w, http.StatusBadRequest, "date query parameter is required")
		return
	}
	date, err := localDate(dateParam, h.location)
	if err != nil {
		writeError(w, http.StatusBadRequest, "date must use YYYY-MM-DD")
		return
	}

	duration := int64(60)
	if serviceID, ok, valid := intQuery(r, "service_id"); ok {
		if !valid || serviceID <= 0 {
			writeError(w, http.StatusBadRequest, "service_id must be a positive integer")
			return
		}
		serviceRow, err := h.queries.GetServiceByID(r.Context(), serviceID)
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
		duration = serviceRow.Minutes
	}

	schedule, err := h.queries.GetScheduleByDay(r.Context(), int64(date.Weekday()))
	if err != nil {
		h.logger.Error("get schedule by day", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	dayStart, dayEnd := dayBounds(date)
	booked, err := h.queries.ListBookingsByDateRange(r.Context(), db.ListBookingsByDateRangeParams{
		DayStart: dayStart,
		DayEnd:   dayEnd,
	})
	if err != nil {
		h.logger.Error("list bookings by date range", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	slots := service.Calculate(date, schedule, booked, duration)
	if slots == nil {
		slots = []service.TimeSlot{}
	}
	writeJSON(w, http.StatusOK, slots)
}
