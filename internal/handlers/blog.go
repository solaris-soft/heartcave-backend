package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gosimple/slug"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/render"
)

// CreatePostRequest is a request to create a blog post
type CreatePostRequest struct {
	// Use the GetSlug method to ensure URL safe
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Published int64  `json:"published"`
}

// Validate returns a map of errors where the key is the field
// in the struct and the value is the error string safe to return
// to the user
func (c CreatePostRequest) Validate() (errors map[string]string) {
	errors = map[string]string{}

	if strings.TrimSpace(c.Slug) == "" {
		errors["slug"] = "slug cannot be empty"
	}

	if strings.TrimSpace(c.Title) == "" {
		errors["title"] = "title cannot be empty"
	}

	if strings.TrimSpace(c.Body) == "" {
		errors["body"] = "body cannot be empty"
	}

	return errors
}

// GetSlug returns a URL safe slug
func (c CreatePostRequest) GetSlug() string {
	return slug.Make(c.Slug)
}

// BlogHandler is the controller for the blog posts resource
type BlogHandler struct {
	queries *db.Queries
	logger  *slog.Logger
}

// NewBlogHandler is the factory for blog handlers
func NewBlogHandler(queries *db.Queries, logger *slog.Logger) BlogHandler {
	return BlogHandler{
		queries: queries,
		logger:  logger,
	}
}

// GetBlogPosts is the handler for retrieving all blog posts
func (b BlogHandler) GetBlogPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := b.queries.ListAllPosts(r.Context())
	if err != nil {
		render.WriteError(w, http.StatusInternalServerError, "Something went wrong.")
		return
	}

	if posts == nil {
		render.WriteJson(w, http.StatusOK, "No posts found")
		return
	}

	render.WriteJson(w, http.StatusOK, posts)
}

// GetBlogPostByID is the handler for retrieving a single blog post by its ID
func (b BlogHandler) GetBlogPostByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		render.WriteError(w, http.StatusBadRequest, "No id parameter provided.")
		return
	}

	intID, err := strconv.Atoi(id)
	if err != nil {
		render.WriteError(w, http.StatusBadRequest, "ID must be an integer")
		return
	}

	post, err := b.queries.GetPostByID(r.Context(), int64(intID))
	if err != nil {
		render.WriteError(w, http.StatusInternalServerError, "Something went wront.")
		return
	}

	render.WriteJson(w, http.StatusOK, post)
}

// CreateBlogPost is the handler for creating a new blog post
func (b BlogHandler) CreateBlogPost(w http.ResponseWriter, r *http.Request) {
	var post CreatePostRequest

	// Decode the json request
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		render.WriteError(w, http.StatusBadRequest, "Invalid request.")
		return
	}

	// Validate the request
	errors := post.Validate()
	if len(errors) != 0 {
		render.WriteValidationErrors(w, errors)
		return
	}

	params := db.CreatePostParams{
		Slug:      post.GetSlug(),
		Title:     post.Title,
		Published: post.Published,
	}

	created, err := b.queries.CreatePost(r.Context(), params)
	if err != nil {
		render.WriteError(w, http.StatusBadRequest, "Invalid request.")
		return
	}

	err = render.WriteJson(w, http.StatusCreated, created)
	if err != nil {
		b.logger.Error("[BLOG] unable to write json response", err)
	}
}
