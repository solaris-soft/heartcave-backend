package main

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/solaris-soft/heartcave-backend/internal/config"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/handler"
	"github.com/solaris-soft/heartcave-backend/internal/handler/api"
	"github.com/solaris-soft/heartcave-backend/internal/render"
	"github.com/solaris-soft/heartcave-backend/internal/router"
	"github.com/solaris-soft/heartcave-backend/web"
	_ "modernc.org/sqlite"
)

func main() {
	cfg := config.New()

	database := cfg.Database.Prepare()
	defer database.Close()

	if err := runMigrations(database); err != nil {
		log.Fatal(err)
	}

	queries := db.New(database)

	tmplFS, err := fs.Sub(web.Templates, "templates")
	if err != nil {
		log.Fatal(err)
	}

	renderer := render.New(tmplFS, []string{
		"blog/list.html",
		"blog/post.html",
		"bookings/form.html",
		"admin/login.html",
		"admin/schedule.html",
		"admin/blog/list.html",
		"admin/blog/form.html",
		"admin/bookings/list.html",
	})

	// Build the auth-layered router topology.
	app := router.New(cfg.Sessions, cfg.JWTSecret)

	// ── Public HTML ──────────────────────────────────────────────────────────
	blog := &handler.BlogHandler{Queries: queries, Renderer: renderer}
	app.Public.Mount("/blog", blog.PublicRoutes())
	app.Public.Get("/book", func(w http.ResponseWriter, r *http.Request) {
		renderer.Page(w, r, "bookings/form.html", nil)
	})

	// ── Admin auth (login / logout — no session required) ────────────────────
	adminAuth := &handler.AdminAuthHandler{Config: cfg, Renderer: renderer}
	app.AdminOpen.Mount("/", adminAuth.Routes())

	// ── Admin panel (session-protected) ──────────────────────────────────────
	app.Admin.Mount("/blog", blog.AdminRoutes())
	app.Admin.Mount("/bookings", (&handler.BookingHandler{Queries: queries, Renderer: renderer}).AdminRoutes())
	app.Admin.Mount("/schedule", (&handler.ScheduleHandler{Queries: queries, Renderer: renderer}).AdminRoutes())

	// ── Public JSON API ───────────────────────────────────────────────────────
	app.API.Mount("/blog", (&api.ApiBlogHandler{Queries: queries}).Routes())
	app.API.Mount("/availability", (&api.ApiAvailabilityHandler{Queries: queries}).Routes())
	app.API.Mount("/auth", (&api.ApiAuthHandler{Queries: queries, JWTSecret: cfg.JWTSecret}).Routes())

	// ── Customer-authenticated API ────────────────────────────────────────────
	app.APIAuth.Mount("", (&api.ApiBookingsHandler{Queries: queries}).Routes())

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, app.HTTP))
}
