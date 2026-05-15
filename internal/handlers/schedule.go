package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/db"
)

type ScheduleHandler struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewScheduleHandler(queries *db.Queries, logger *slog.Logger) ScheduleHandler {
	return ScheduleHandler{queries: queries, logger: logger}
}

type scheduleRequest struct {
	DayOfWeek   int64  `json:"dayOfWeek"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	SlotMinutes int64  `json:"slotMinutes"`
}

func (h ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	schedule, err := h.queries.ListSchedule(r.Context())
	if err != nil {
		h.logger.Error("list schedule", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if schedule == nil {
		schedule = []db.AdminSchedule{}
	}
	writeJSON(w, http.StatusOK, schedule)
}

func (h ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req scheduleRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	params, ok := scheduleParams(req)
	if !ok {
		writeError(w, http.StatusBadRequest, "valid dayOfWeek, startTime, endTime, and slotMinutes are required")
		return
	}
	entry, err := h.queries.CreateScheduleEntry(r.Context(), params)
	if err != nil {
		h.logger.Error("create schedule", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (h ScheduleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := routeID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req scheduleRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	params, ok := scheduleParams(req)
	if !ok {
		writeError(w, http.StatusBadRequest, "valid dayOfWeek, startTime, endTime, and slotMinutes are required")
		return
	}
	entry, err := h.queries.UpdateScheduleEntry(r.Context(), db.UpdateScheduleEntryParams{
		DayOfWeek:   params.DayOfWeek,
		StartTime:   params.StartTime,
		EndTime:     params.EndTime,
		SlotMinutes: params.SlotMinutes,
		ID:          id,
	})
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := routeID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.queries.DeleteScheduleEntry(r.Context(), id); err != nil {
		h.logger.Error("delete schedule", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func scheduleParams(req scheduleRequest) (db.CreateScheduleEntryParams, bool) {
	if req.DayOfWeek < 0 || req.DayOfWeek > 6 || req.SlotMinutes <= 0 {
		return db.CreateScheduleEntryParams{}, false
	}
	start, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		return db.CreateScheduleEntryParams{}, false
	}
	end, err := time.Parse("15:04", req.EndTime)
	if err != nil || !end.After(start) {
		return db.CreateScheduleEntryParams{}, false
	}
	return db.CreateScheduleEntryParams{
		DayOfWeek:   req.DayOfWeek,
		StartTime:   start.Format("15:04"),
		EndTime:     end.Format("15:04"),
		SlotMinutes: req.SlotMinutes,
	}, true
}
