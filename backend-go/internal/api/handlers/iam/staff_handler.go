package iam

import (
	"encoding/json"
	"net/http"

	iamDto "github.com/google-hackathon/rapid-response/internal/api/dto/iam"
	iamService "github.com/google-hackathon/rapid-response/internal/service/iam"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type StaffHandler struct {
	service *iamService.StaffService
}

func NewStaffHandler(service *iamService.StaffService) *StaffHandler {
	return &StaffHandler{service: service}
}

func (h *StaffHandler) BulkCreateStaff(w http.ResponseWriter, r *http.Request) {
	hotelIDStr := chi.URLParam(r, "hotel_id")
	hotelID, err := uuid.Parse(hotelIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid hotel_id"})
		return
	}

	var req iamDto.BulkCreateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	resp, err := h.service.BulkCreateStaff(r.Context(), hotelID, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
