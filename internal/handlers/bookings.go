package handlers

import (
	"net/http"

	"github.com/solaris-soft/heartcave-backend/internal/database"
)

type BookingsHandler struct {
	DB database.Querier
}

func (h BookingsHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
}
