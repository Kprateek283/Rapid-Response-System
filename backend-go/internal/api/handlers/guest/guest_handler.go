package guest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	guestDto "github.com/google-hackathon/rapid-response/internal/api/dto/guest"
	guestService "github.com/google-hackathon/rapid-response/internal/service/guest"
	"github.com/google/uuid"
)

type GuestHandler struct {
	service *guestService.GuestService
}

func NewGuestHandler(service *guestService.GuestService) *GuestHandler {
	return &GuestHandler{service: service}
}

func (h *GuestHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	hotelIDStr := chi.URLParam(r, "hotel_id")
	hotelID, err := uuid.Parse(hotelIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid hotel_id"})
		return
	}

	roomIDStr := chi.URLParam(r, "room_id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid room_id"})
		return
	}

	var req guestDto.CheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	resp, err := h.service.CheckIn(r.Context(), hotelID, roomID, req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "room is already occupied" {
			status = http.StatusConflict
		} else if err.Error() == "room not found" {
			status = http.StatusNotFound
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *GuestHandler) CheckOut(w http.ResponseWriter, r *http.Request) {
	hotelIDStr := chi.URLParam(r, "hotel_id")
	hotelID, err := uuid.Parse(hotelIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid hotel_id"})
		return
	}

	roomIDStr := chi.URLParam(r, "room_id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid room_id"})
		return
	}

	resp, err := h.service.CheckOut(r.Context(), hotelID, roomID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "room is not currently occupied" {
			status = http.StatusBadRequest
		} else if err.Error() == "room not found" {
			status = http.StatusNotFound
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
