package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/render"
)

// BlogHandler serves both the public blog pages and the admin blog management UI.
type BlogHandler struct {
	Queries  *db.Queries
	Renderer *render.Renderer
}

// PublicRoutes returns the public-facing blog sub-router.
// Mount at /blog on the public router.
func (h *BlogHandler) PublicRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{slug}", h.Show)
	return r
}

// AdminRoutes returns the admin blog management sub-router.
// Mount at /blog on the protected admin router.
func (h *BlogHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.AdminList)
	r.Get("/new", h.AdminNew)
	r.Post("/", h.AdminCreate)
	r.Get("/{id}/edit", h.AdminEdit)
	r.Post("/{id}", h.AdminUpdate)
	r.Post("/{id}/delete", h.AdminDelete)
	return r
}

// List renders the public blog post listing.
func (h *BlogHandler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := h.Queries.ListPublishedPosts(r.Context())
	if err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	h.Renderer.Page(w, r, "blog/list.html", posts)
}

// Show renders a single published post.
func (h *BlogHandler) Show(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	post, err := h.Queries.GetPostBySlug(r.Context(), slug)
	if err != nil {
		h.Renderer.NotFound(w, r)
		return
	}
	h.Renderer.Page(w, r, "blog/post.html", post)
}

// AdminList renders all posts (published and draft) for the admin.
func (h *BlogHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	posts, err := h.Queries.ListAllPosts(r.Context())
	if err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	h.Renderer.Page(w, r, "admin/blog/list.html", posts)
}

// AdminNew renders the new post form.
func (h *BlogHandler) AdminNew(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Page(w, r, "admin/blog/form.html", nil)
}

// AdminCreate handles the new post form submission.
func (h *BlogHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.BadRequest(w)
		return
	}

	published := int64(0)
	if r.FormValue("published") == "on" {
		published = 1
	}

	_, err := h.Queries.CreatePost(r.Context(), db.CreatePostParams{
		Slug:      r.FormValue("slug"),
		Title:     r.FormValue("title"),
		Body:      r.FormValue("body"),
		Published: published,
	})
	if err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}

// AdminEdit renders the edit form for an existing post.
func (h *BlogHandler) AdminEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.Renderer.NotFound(w, r)
		return
	}
	post, err := h.Queries.GetPostByID(r.Context(), id)
	if err != nil {
		h.Renderer.NotFound(w, r)
		return
	}
	h.Renderer.Page(w, r, "admin/blog/form.html", post)
}

// AdminUpdate handles the edit form submission.
func (h *BlogHandler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.Renderer.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.Renderer.BadRequest(w)
		return
	}

	published := int64(0)
	if r.FormValue("published") == "on" {
		published = 1
	}

	_, err = h.Queries.UpdatePost(r.Context(), db.UpdatePostParams{
		ID:        id,
		Slug:      r.FormValue("slug"),
		Title:     r.FormValue("title"),
		Body:      r.FormValue("body"),
		Published: published,
	})
	if err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}

// AdminDelete deletes a post.
func (h *BlogHandler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.Renderer.NotFound(w, r)
		return
	}
	if err := h.Queries.DeletePost(r.Context(), id); err != nil {
		h.Renderer.ServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}
