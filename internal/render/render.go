// Package render provides pre-parsed template caching and response helpers.
// Each page template is paired with base.html once at startup; subsequent
// renders are pure Execute calls with no I/O or parsing overhead.
package render

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
)

// Renderer holds a pre-built *template.Template for every registered page.
// Each entry is an isolated parse set (base.html + one page), so Go template
// block definitions never collide across pages.
type Renderer struct {
	cache map[string]*template.Template
}

// New parses base.html paired with each path in pages and caches the results.
// It panics on any parse error so misconfiguration is caught at startup.
func New(tmplFS fs.FS, pages []string) *Renderer {
	cache := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		t := template.Must(template.ParseFS(tmplFS, "base.html", page))
		cache[page] = t
	}
	return &Renderer{cache: cache}
}

// Page renders the named page by executing the "base" template from the
// pre-built cache. If the page is unknown or execution fails it logs the
// error and writes an appropriate HTTP error response.
func (re *Renderer) Page(w http.ResponseWriter, r *http.Request, page string, data any) {
	t, ok := re.cache[page]
	if !ok {
		log.Printf("render: unknown template %q", page)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		// Headers are already written; log and move on.
		log.Printf("render: execute %q: %v", page, err)
	}
}

// ServerError writes a 500 response. Call before any other writes on w.
func (re *Renderer) ServerError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// NotFound writes a 404 response.
func (re *Renderer) NotFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// BadRequest writes a 400 response.
func (re *Renderer) BadRequest(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}
