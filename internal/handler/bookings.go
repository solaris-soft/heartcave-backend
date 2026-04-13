package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/render"
)

// BookingHandler serves admin booking management pages.
type BookingHandler struct {
	Queries  *db.Queries
	Renderer *render.Renderer
}

// AdminRoutes returns the admin bookings sub-router.
// Mount at /bookings on the protected admin router.
func (h *BookingHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.AdminList)
	r.Post("/{id}/status", h.AdminUpdateStatus)
	return r
}

// AdminList renders all bookings for the admin.
func (h *BookingHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.Queries.ListBookings(r.Context())
	if err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	h.Renderer.Page(w, r, "admin/bookings/list.html", bookings)
}

// AdminUpdateStatus updates the status of a booking via a form POST.
func (h *BookingHandler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.Renderer.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.Renderer.BadRequest(w)
		return
	}

	err = h.Queries.UpdateBookingStatus(r.Context(), db.UpdateBookingStatusParams{
		ID:     id,
		Status: r.FormValue("status"),
	})
	if err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/bookings", http.StatusSeeOther)
}
