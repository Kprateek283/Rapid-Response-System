package iam

import (
	"encoding/json"
	"net/http"

	iamDto "github.com/google-hackathon/rapid-response/internal/api/dto/iam"
	iamService "github.com/google-hackathon/rapid-response/internal/service/iam"
)

type HotelHandler struct {
	service *iamService.HotelService
}

func NewHotelHandler(service *iamService.HotelService) *HotelHandler {
	return &HotelHandler{service: service}
}

func (h *HotelHandler) CreateHotel(w http.ResponseWriter, r *http.Request) {
	var req iamDto.CreateHotelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	resp, err := h.service.CreateHotel(r.Context(), req)
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
