package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gosimple/slug"
	"github.com/solaris-soft/heartcave-backend/internal/db"
)

type BlogHandler struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewBlogHandler(queries *db.Queries, logger *slog.Logger) BlogHandler {
	return BlogHandler{queries: queries, logger: logger}
}

type postRequest struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Published bool   `json:"published"`
}

type publicPostListResponse struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	PublishedAt string `json:"published_at"`
}

type publicPostResponse struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

func (h BlogHandler) ListPublished(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	posts, err := h.queries.ListPublishedPosts(r.Context(), db.ListPublishedPostsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("list published posts", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if posts == nil {
		posts = []db.ListPublishedPostsRow{}
	}
	response := make([]publicPostListResponse, len(posts))
	for i, post := range posts {
		response[i] = publicPostListResponse{
			Slug:        post.Slug,
			Title:       post.Title,
			PublishedAt: post.CreatedAt,
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (h BlogHandler) GetPublishedBySlug(w http.ResponseWriter, r *http.Request) {
	post, err := h.queries.GetPostBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicPostResponse{
		Slug:        post.Slug,
		Title:       post.Title,
		Body:        post.Body,
		PublishedAt: post.CreatedAt,
	})
}

func (h BlogHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	posts, err := h.queries.ListAllPosts(r.Context())
	if err != nil {
		h.logger.Error("list all posts", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if posts == nil {
		posts = []db.Post{}
	}
	writeJSON(w, http.StatusOK, posts)
}

func (h BlogHandler) AdminGet(w http.ResponseWriter, r *http.Request) {
	id, ok := routeID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	post, err := h.queries.GetPostByID(r.Context(), id)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (h BlogHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var req postRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	params, ok := postParams(req)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errors": map[string]string{
			"title": "title is required",
			"body":  "body is required",
		}})
		return
	}

	post, err := h.queries.CreatePost(r.Context(), params)
	if err != nil {
		h.logger.Error("create post", "err", err)
		writeError(w, http.StatusConflict, "slug already exists")
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

func (h BlogHandler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := routeID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req postRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	params, ok := postParams(req)
	if !ok {
		writeError(w, http.StatusBadRequest, "title and body are required")
		return
	}

	post, err := h.queries.UpdatePost(r.Context(), db.UpdatePostParams{
		Slug:      params.Slug,
		Title:     params.Title,
		Body:      params.Body,
		Published: params.Published,
		ID:        id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.logger.Error("update post", "err", err)
		writeError(w, http.StatusConflict, "slug already exists")
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (h BlogHandler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := routeID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.queries.DeletePost(r.Context(), id); err != nil {
		h.logger.Error("delete post", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func postParams(req postRequest) (db.CreatePostParams, bool) {
	title := strings.TrimSpace(req.Title)
	body := strings.TrimSpace(req.Body)
	if title == "" || body == "" {
		return db.CreatePostParams{}, false
	}
	rawSlug := strings.TrimSpace(req.Slug)
	if rawSlug == "" {
		rawSlug = title
	}
	published := int64(0)
	if req.Published {
		published = 1
	}
	return db.CreatePostParams{
		Slug:      slug.Make(rawSlug),
		Title:     title,
		Body:      body,
		Published: published,
	}, true
}
