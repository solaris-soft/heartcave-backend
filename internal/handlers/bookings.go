package handlers

import (
	"log/slog"

	"github.com/solaris-soft/heartcave-backend/internal/db"
)

type BookingsHandler struct {
	queries *db.Queries
	logger  *slog.Logger
}
