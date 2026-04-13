package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/config"
	"github.com/solaris-soft/heartcave-backend/internal/render"
)

// AdminAuthHandler handles admin login and logout.
type AdminAuthHandler struct {
	Config   *config.Config
	Renderer *render.Renderer
}

// Routes returns the login/logout sub-router.
// Mount this on the open (unauthenticated) /admin router.
func (h *AdminAuthHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/login", h.LoginPage)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	return r
}

// LoginPage renders the admin login form.
func (h *AdminAuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Page(w, r, "admin/login.html", nil)
}

// Login validates credentials and creates an admin session.
func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.BadRequest(w)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email != h.Config.AdminEmail || password != h.Config.AdminPassword {
		h.Renderer.Page(w, r, "admin/login.html", map[string]string{"Error": "Invalid credentials"})
		return
	}

	h.Config.Sessions.Put(r.Context(), "admin", true)
	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}

// Logout destroys the admin session.
func (h *AdminAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.Config.Sessions.Destroy(r.Context())
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
