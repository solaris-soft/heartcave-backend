// Package router configures the HTTP routing topology and middleware layers.
// It owns the auth boundary decisions (which paths require which auth) but
// knows nothing about specific handler types — callers mount sub-handlers via
// the exposed chi.Router fields.
package router

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	appmiddleware "github.com/solaris-soft/heartcave-backend/internal/middleware"
)

// App holds the root HTTP handler plus the auth-stratified sub-routers
// that callers mount their handlers onto.
type App struct {
	// HTTP is the root handler to pass to http.ListenAndServe.
	HTTP http.Handler

	// Public is the base router — no auth required.
	Public chi.Router

	// Admin is the /admin sub-router protected by session auth.
	// Mount admin panel handlers here.
	Admin chi.Router

	// AdminOpen is the /admin sub-router that is NOT protected by session auth.
	// Use this only for the login/logout endpoints.
	AdminOpen chi.Router

	// API is the /api sub-router — no auth required.
	// Mount public API handlers here.
	API chi.Router

	// APIAuth is the /api/me sub-router protected by JWT auth.
	// Mount customer-facing authenticated API handlers here.
	APIAuth chi.Router
}

// New builds the middleware topology and returns the App with its mountable
// sub-routers. Callers register their own handlers on the exposed routers.
func New(sessions *scs.SessionManager, jwtSecret string) *App {
	root := chi.NewRouter()
	root.Use(middleware.Logger)
	root.Use(middleware.Recoverer)
	root.Use(sessions.LoadAndSave)

	// /admin — login/logout accessible before session check
	adminOpen := chi.NewRouter()
	root.Mount("/admin", adminOpen)

	// /admin — protected area
	adminProtected := chi.NewRouter()
	adminProtected.Use(appmiddleware.AdminRequired(sessions))
	adminOpen.Mount("/", adminProtected)

	// /api — public
	apiPublic := chi.NewRouter()
	root.Mount("/api", apiPublic)

	// /api/me — customer JWT protected
	apiAuth := chi.NewRouter()
	apiAuth.Use(appmiddleware.CustomerRequired(jwtSecret))
	apiPublic.Mount("/me", apiAuth)

	return &App{
		HTTP:      root,
		Public:    root,
		Admin:     adminProtected,
		AdminOpen: adminOpen,
		API:       apiPublic,
		APIAuth:   apiAuth,
	}
}
