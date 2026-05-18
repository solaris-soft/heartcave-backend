package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
)

type ServiceAvailabilityHandler struct {
	DB database.Querier
}

func NewServiceAvailabilityHandler(db database.Querier) ServiceAvailabilityHandler {
	return ServiceAvailabilityHandler{db}
}

type CreateAvailabilityRequest struct {
	ServiceID uuid.UUID `json:"service_id"`
	DayOfWeek int16     `json:"day_of_week"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (c CreateAvailabilityRequest) IsValidTimes() bool {
	return c.EndTime.After(c.StartTime)
}

func (c CreateAvailabilityRequest) IsValidWeekDay() bool {
	return c.DayOfWeek >= 0 && c.DayOfWeek <= 6
}

func (h ServiceAvailabilityHandler) CreateAvailability(w http.ResponseWriter, r *http.Request) {
	var req CreateAvailabilityRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	if !req.IsValidTimes() || !req.IsValidWeekDay() {
		WriteBadRequest(w)
		return
	}
	availability, err := h.DB.CreateAvailability(r.Context(), database.CreateAvailabilityParams{
		ServiceID: req.ServiceID,
		DayOfWeek: req.DayOfWeek,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	})
	if err != nil {
		WriteInternalError(w)
		return
	}
	WriteJSON(w, http.StatusCreated, availability)
}

func (h ServiceAvailabilityHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
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
	availability, err := h.DB.GetAvailabilityByID(r.Context(), parsedID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		WriteInternalError(w)
		return
	}
	WriteJSON(w, http.StatusOK, availability)
}

func (h ServiceAvailabilityHandler) GetAllAvailability(w http.ResponseWriter, r *http.Request) {
	availabilities, err := h.DB.GetAllAvailability(r.Context())
	if err != nil {
		WriteInternalError(w)
		return
	}
	WriteJSON(w, http.StatusOK, availabilities)
}

func (h ServiceAvailabilityHandler) GetAvailabilityByService(w http.ResponseWriter, r *http.Request) {
	serviceID := r.PathValue("service_id")
	if serviceID == "" {
		WriteBadRequest(w)
		return
	}
	parsedID, err := uuid.Parse(serviceID)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	availabilities, err := h.DB.GetAvailabilityByServiceID(r.Context(), parsedID)
	if err != nil {
		WriteInternalError(w)
		return
	}
	WriteJSON(w, http.StatusOK, availabilities)
}

type UpdateAvailabilityRequest struct {
	ServiceID *uuid.UUID `json:"service_id"`
	DayOfWeek *int16     `json:"day_of_week"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

func (u UpdateAvailabilityRequest) IsEmpty() bool {
	return u.ServiceID == nil &&
		u.DayOfWeek == nil &&
		u.StartTime == nil &&
		u.EndTime == nil
}

func (h ServiceAvailabilityHandler) UpdateAvailability(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	var req UpdateAvailabilityRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	if req.IsEmpty() {
		WriteBadRequest(w)
		return
	}
	original, err := h.DB.GetAvailabilityByID(r.Context(), parsedID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		WriteInternalError(w)
		return
	}
	params := database.UpdateAvailabilityByIDParams{
		ID:        original.ID,
		ServiceID: original.ServiceID,
		DayOfWeek: original.DayOfWeek,
		StartTime: original.StartTime,
		EndTime:   original.EndTime,
	}
	if req.ServiceID != nil {
		params.ServiceID = *req.ServiceID
	}
	if req.DayOfWeek != nil {
		params.DayOfWeek = *req.DayOfWeek
	}
	if req.StartTime != nil {
		params.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		params.EndTime = *req.EndTime
	}
	if !params.EndTime.After(params.StartTime) {
		WriteBadRequest(w)
		return
	}
	if params.DayOfWeek < 0 || params.DayOfWeek > 6 {
		WriteBadRequest(w)
		return
	}
	updated, err := h.DB.UpdateAvailabilityByID(r.Context(), params)
	if err != nil {
		WriteInternalError(w)
		return
	}
	WriteJSON(w, http.StatusOK, updated)
}

func (h ServiceAvailabilityHandler) DeleteAvailability(w http.ResponseWriter, r *http.Request) {
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
	err = h.DB.DeleteAvailabilityByID(r.Context(), parsedID)
	if err != nil {
		WriteInternalError(w)
		return
	}
	w.WriteHeader(http.StatusOK)
}
