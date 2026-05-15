package handlers

import (
	"log/slog"
	"net/http"

	"github.com/solaris-soft/heartcave-backend/internal/db"
)

type CustomersHandler struct {
	queries *db.Queries
	logger  *slog.Logger
}

type customerResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func NewCustomersHandler(queries *db.Queries, logger *slog.Logger) CustomersHandler {
	return CustomersHandler{queries: queries, logger: logger}
}

func (h CustomersHandler) List(w http.ResponseWriter, r *http.Request) {
	customers, err := h.queries.ListCustomers(r.Context())
	if err != nil {
		h.logger.Error("list customers", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if customers == nil {
		customers = []db.Customer{}
	}
	response := make([]customerResponse, len(customers))
	for i, customer := range customers {
		response[i] = customerResponse{
			ID:        customer.ID,
			Name:      customer.Name,
			Email:     customer.Email,
			Phone:     customer.Phone,
			CreatedAt: customer.CreatedAt,
			UpdatedAt: customer.UpdatedAt,
		}
	}
	writeJSON(w, http.StatusOK, response)
}
