package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/db"
)

// ApiBlogHandler serves published blog posts as JSON.
type ApiBlogHandler struct {
	Queries *db.Queries
}

// Routes returns the blog API sub-router.
// Mount at /blog on the API router.
func (h *ApiBlogHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{slug}", h.Show)
	return r
}

type blogListItem struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	CreatedAt string `json:"published_at"`
}

type blogPost struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"published_at"`
}

// List returns all published posts (slug, title, published_at).
func (h *ApiBlogHandler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := h.Queries.ListPublishedPosts(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	items := make([]blogListItem, len(posts))
	for i, p := range posts {
		items[i] = blogListItem{Slug: p.Slug, Title: p.Title, CreatedAt: p.CreatedAt}
	}
	writeJSON(w, http.StatusOK, items)
}

// Show returns a single published post by slug.
func (h *ApiBlogHandler) Show(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	post, err := h.Queries.GetPostBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, blogPost{
		Slug:      post.Slug,
		Title:     post.Title,
		Body:      post.Body,
		CreatedAt: post.CreatedAt,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
