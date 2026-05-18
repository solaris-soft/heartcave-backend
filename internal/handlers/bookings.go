package handlers

import (
	"net/http"

	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/solaris-soft/heartcave-backend/internal/services"
)

type BookingsHandler struct {
	DB             database.Querier
	BookingService services.BookingService
}

func (h BookingsHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
}
