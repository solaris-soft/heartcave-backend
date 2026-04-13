package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/render"
)

// ScheduleHandler manages the admin's weekly availability schedule.
type ScheduleHandler struct {
	Queries  *db.Queries
	Renderer *render.Renderer
}

// AdminRoutes returns the admin schedule management sub-router.
// Mount at /schedule on the protected admin router.
func (h *ScheduleHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.AdminList)
	r.Post("/", h.AdminCreate)
	r.Post("/{id}/delete", h.AdminDelete)
	return r
}

// AdminList renders the schedule management page.
func (h *ScheduleHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	entries, err := h.Queries.ListSchedule(r.Context())
	if err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	h.Renderer.Page(w, r, "admin/schedule.html", entries)
}

// AdminCreate adds a new schedule entry.
func (h *ScheduleHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.BadRequest(w)
		return
	}

	dayOfWeek, err := strconv.ParseInt(r.FormValue("day_of_week"), 10, 64)
	if err != nil {
		h.Renderer.BadRequest(w)
		return
	}
	slotMinutes, err := strconv.ParseInt(r.FormValue("slot_minutes"), 10, 64)
	if err != nil {
		slotMinutes = 60
	}

	_, err = h.Queries.CreateScheduleEntry(r.Context(), db.CreateScheduleEntryParams{
		DayOfWeek:   dayOfWeek,
		StartTime:   r.FormValue("start_time"),
		EndTime:     r.FormValue("end_time"),
		SlotMinutes: slotMinutes,
	})
	if err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/schedule", http.StatusSeeOther)
}

// AdminDelete removes a schedule entry.
func (h *ScheduleHandler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.Renderer.NotFound(w, r)
		return
	}
	if err := h.Queries.DeleteScheduleEntry(r.Context(), id); err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/schedule", http.StatusSeeOther)
}
