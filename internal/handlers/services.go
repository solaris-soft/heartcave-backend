package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
)

// ServicesHandler is the handler for Admins to manage the services resource
type ServicesHandler struct {
	DB database.Querier
}

func NewServicesHandler(db database.Querier) ServicesHandler {
	return ServicesHandler{
		DB: db,
	}
}

// CreateService allows admins to create a service
func (h ServicesHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	type CreateServiceRequest struct {
		Name           string `json:"name"`
		Cents          int    `json:"cents"`
		Description    string `json:"description"`
		SessionMinutes int    `json:"session_minutes"`
	}
	var req CreateServiceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	if req.SessionMinutes <= 0 {
		WriteBadRequest(w)
		return
	}
	priceStr := fmt.Sprintf("%.2f", float64(req.Cents)/100.0)
	service, err := h.DB.CreateService(r.Context(), database.CreateServiceParams{
		Name:           req.Name,
		Price:          priceStr,
		Description:    req.Description,
		SessionMinutes: int32(req.SessionMinutes),
	})
	if err != nil {
		WriteInternalError(w)
		return
	}

	WriteJSON(w, http.StatusCreated, service)
}

type UpdateServiceRequest struct {
	Name           *string `json:"name"`
	Price          *int    `json:"cents"`
	Description    *string `json:"description"`
	SessionMinutes *int32  `json:"session_minutes"`
}

func (u UpdateServiceRequest) IsEmpty() bool {
	return u.Name == nil &&
		u.Price == nil &&
		u.Description == nil &&
		u.SessionMinutes == nil
}

func (u UpdateServiceRequest) MakeUpdateParams(original database.Service) database.UpdateServiceByIDParams {
	params := database.UpdateServiceByIDParams{
		ID:             original.ID,
		Name:           original.Name,
		Price:          original.Price,
		Description:    original.Description,
		SessionMinutes: original.SessionMinutes,
	}
	if u.Name != nil {
		params.Name = *u.Name
	}
	if u.Description != nil {
		params.Description = *u.Description
	}
	if u.SessionMinutes != nil {
		params.SessionMinutes = *u.SessionMinutes
	}
	if u.Price != nil {
		params.Price = fmt.Sprintf("%.2f", float64(*u.Price)/100.0)
	}
	return params
}

func (h ServicesHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	idParsed, err := uuid.Parse(id)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	var req UpdateServiceRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	if req.IsEmpty() {
		WriteBadRequest(w)
		return
	}
	service, err := h.DB.GetServiceByID(r.Context(), idParsed)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	params := req.MakeUpdateParams(service)
	updatedService, err := h.DB.UpdateServiceByID(r.Context(), params)
	if err != nil {
		WriteInternalError(w)
		return
	}
	WriteJSON(w, http.StatusOK, updatedService)
}

func (h ServicesHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		WriteBadRequest(w)
		return
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	err = h.DB.DeleteServiceByID(r.Context(), parsedID)
	if err != nil {
		WriteInternalError(w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h ServicesHandler) GetServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.DB.GetAllServices(r.Context())
	if err != nil {
		WriteInternalError(w)
		return
	}
	WriteJSON(w, http.StatusOK, services)
}

func (h ServicesHandler) GetService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		WriteBadRequest(w)
		return
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	service, err := h.DB.GetServiceByID(r.Context(), parsedID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		WriteInternalError(w)
		return
	}
	WriteJSON(w, http.StatusOK, service)
}
