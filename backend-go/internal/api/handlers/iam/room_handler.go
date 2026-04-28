package iam

import (
	"encoding/json"
	"net/http"
	"strings"

	iamDto "github.com/google-hackathon/rapid-response/internal/api/dto/iam"
	iamService "github.com/google-hackathon/rapid-response/internal/service/iam"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RoomHandler struct {
	service *iamService.RoomService
}

func NewRoomHandler(service *iamService.RoomService) *RoomHandler {
	return &RoomHandler{service: service}
}

func (h *RoomHandler) BulkCreateRooms(w http.ResponseWriter, r *http.Request) {
	hotelIDStr := chi.URLParam(r, "hotel_id")
	hotelID, err := uuid.Parse(hotelIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid hotel_id"})
		return
	}

	var req iamDto.BulkCreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	resp, err := h.service.BulkCreateRooms(r.Context(), hotelID, req)
	if err != nil {
		status := http.StatusInternalServerError
		errorCode := ""
		// Check for duplicate entry errors (unique constraint violation)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "23505") {
			status = http.StatusConflict
			errorCode = "DUPLICATE_ENTRY"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		resp := map[string]string{"status": "error", "message": err.Error()}
		if errorCode != "" {
			resp["error_code"] = errorCode
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
