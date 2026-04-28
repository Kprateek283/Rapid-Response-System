package onboard

import (
	"encoding/json"
	"net/http"

	"github.com/google-hackathon/rapid-response/internal/api/middleware"
	onboardDto "github.com/google-hackathon/rapid-response/internal/api/dto/onboard"
	"github.com/google-hackathon/rapid-response/internal/models"
	onboardService "github.com/google-hackathon/rapid-response/internal/service/onboard"
)

type OnboardHandler struct {
	service *onboardService.OnboardService
}

func NewOnboardHandler(service *onboardService.OnboardService) *OnboardHandler {
	return &OnboardHandler{service: service}
}

// Scan handles QR code scanning — public endpoint validated by qr_index + is_occupied.
func (h *OnboardHandler) Scan(w http.ResponseWriter, r *http.Request) {
	var req onboardDto.ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	resp, err := h.service.Scan(r.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "invalid QR code: room not found" {
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

// PairStatus handles BLE pairing confirmation — requires GUEST JWT.
func (h *OnboardHandler) PairStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok || claims.GuestID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Guest token required"})
		return
	}

	var req onboardDto.PairStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	resp, err := h.service.PairStatus(r.Context(), claims.GuestID, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
