package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/solaris-soft/heartcave-backend/internal/db"
)

type ServicesHandler struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewServicesHandler(queries *db.Queries, logger *slog.Logger) ServicesHandler {
	return ServicesHandler{queries: queries, logger: logger}
}

type serviceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Minutes     int64  `json:"minutes"`
}

type serviceResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Minutes     int64  `json:"minutes"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (h ServicesHandler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.queries.ListServices(r.Context())
	if err != nil {
		h.logger.Error("list services", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if services == nil {
		services = []db.Service{}
	}
	writeJSON(w, http.StatusOK, serviceResponses(services))
}

func (h ServicesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req serviceRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	params, ok := createServiceParams(req)
	if !ok {
		writeError(w, http.StatusBadRequest, "name, positive price, and positive minutes are required")
		return
	}
	service, err := h.queries.CreateService(r.Context(), params)
	if err != nil {
		h.logger.Error("create service", "err", err)
		writeError(w, http.StatusConflict, "service name already exists")
		return
	}
	writeJSON(w, http.StatusCreated, serviceFromDB(service))
}

func (h ServicesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := routeID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req serviceRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	params, ok := createServiceParams(req)
	if !ok {
		writeError(w, http.StatusBadRequest, "name, positive price, and positive minutes are required")
		return
	}
	service, err := h.queries.UpdateService(r.Context(), db.UpdateServiceParams{
		Name:        params.Name,
		Description: params.Description,
		Price:       params.Price,
		Minutes:     params.Minutes,
		ID:          id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.logger.Error("update service", "err", err)
		writeError(w, http.StatusConflict, "service name already exists")
		return
	}
	writeJSON(w, http.StatusOK, serviceFromDB(service))
}

func (h ServicesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := routeID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.queries.DeleteService(r.Context(), id); err != nil {
		h.logger.Error("delete service", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serviceResponses(services []db.Service) []serviceResponse {
	response := make([]serviceResponse, len(services))
	for i, service := range services {
		response[i] = serviceFromDB(service)
	}
	return response
}

func serviceFromDB(service db.Service) serviceResponse {
	description := ""
	if service.Description.Valid {
		description = service.Description.String
	}
	return serviceResponse{
		ID:          service.ID,
		Name:        service.Name,
		Description: description,
		Price:       service.Price,
		Minutes:     service.Minutes,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}
}

func createServiceParams(req serviceRequest) (db.CreateServiceParams, bool) {
	name := strings.TrimSpace(req.Name)
	if name == "" || req.Price <= 0 || req.Minutes <= 0 {
		return db.CreateServiceParams{}, false
	}
	return db.CreateServiceParams{
		Name:        name,
		Description: nullString(req.Description),
		Price:       req.Price,
		Minutes:     req.Minutes,
	}, true
}
