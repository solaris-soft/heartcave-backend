package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/render"
)

// CreateCustomerRequest represents a request to create new customer
type CreateCustomerRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

func (c CreateCustomerRequest) Validate() (errors map[string]string) {
	errors = map[string]string{}

	if strings.TrimSpace(c.Name) == "" {
		errors["name"] = "name is required"
	}

	if !strings.Contains(c.Email, "@") {
		errors["email"] = "invalid email address"
	}

	if len(c.Password) < 8 {
		errors["password"] = "password must be at least 8 characters"
	}

	if c.Password != c.ConfirmPassword {
		errors["password"] = "password and confirm password do not match."
	}

	return errors
}

// Customer handler handles the customer resource
type CustomerHandler struct {
	queries *db.Queries
	logger  *slog.Logger
}

// NewCustomerHandler is the factory function for a customer handler
func NewCustomerHandler(queries *db.Queries, logger *slog.Logger) CustomerHandler {
	return CustomerHandler{
		queries: queries,
		logger:  logger,
	}
}

// CreateCustomer creates a customer
func (c CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var customerRequest CreateCustomerRequest

	err := json.NewDecoder(r.Body).Decode(&customerRequest)
	if err != nil {
		render.WriteError(w, http.StatusBadRequest, "Invalid request.")
		return
	}

	errors := customerRequest.Validate()
	if len(errors) != 0 {
		render.WriteValidationErrors(w, errors)
		return
	}

	createCustomerParams := db.CreateCustomerParams{
		Name:  customerRequest.Name,
		Email: customerRequest.Phone,
		Phone: customerRequest.Phone,
	}
	customer, err := c.queries.CreateCustomer(r.Context(), createCustomerParams)
	if err != nil {
		render.WriteError(w, http.StatusInternalServerError, "Something went wrong.")
		return
	}

	render.WriteJson(w, http.StatusCreated, customer)
}
