package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/solaris-soft/heartcave-backend/internal/render"
	"github.com/solaris-soft/heartcave-backend/internal/service"
)

func writeError(w http.ResponseWriter, status int, message string) {
	_ = render.WriteError(w, status, message)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	_ = render.WriteJson(w, status, data)
}

func decodeJSON(r *http.Request, dst any) bool {
	return render.DecodeJson(r, dst) == nil
}

func routeID(r *http.Request, name string) (int64, bool) {
	raw := chi.URLParam(r, name)
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil
}

func intQuery(r *http.Request, name string) (int64, bool, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return 0, false, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	return value, true, err == nil
}

func pagination(r *http.Request) (limit int64, offset int64) {
	limit = 20
	if value, ok, valid := intQuery(r, "limit"); ok && valid && value > 0 && value <= 100 {
		limit = value
	}
	if value, ok, valid := intQuery(r, "offset"); ok && valid && value >= 0 {
		offset = value
	}
	return limit, offset
}

func required(value string) bool {
	return strings.TrimSpace(value) != ""
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func localDate(value string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, loc)
}

func localDateTime(value string, loc *time.Location) (time.Time, error) {
	if parsed, err := time.ParseInLocation(service.LocalDateTimeLayout, value, loc); err == nil {
		return parsed, nil
	}
	return time.ParseInLocation("2006-01-02T15:04", value, loc)
}

func dayBounds(date time.Time) (string, string) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	return start.Format(service.LocalDateTimeLayout), start.AddDate(0, 0, 1).Format(service.LocalDateTimeLayout)
}

func notFoundOrServer(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}
