package middleware

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
)

// AdminRequired redirects unauthenticated requests to /admin/login.
func AdminRequired(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !sm.GetBool(r.Context(), "admin") {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
