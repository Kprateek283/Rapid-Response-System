package iam

import (
	"encoding/json"
	"net/http"

	iamDto "github.com/google-hackathon/rapid-response/internal/api/dto/iam"
	iamService "github.com/google-hackathon/rapid-response/internal/service/iam"
)

type GroupHandler struct {
	service *iamService.GroupService
}

func NewGroupHandler(service *iamService.GroupService) *GroupHandler {
	return &GroupHandler{service: service}
}

func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req iamDto.CreateGroupRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"status":"error","message":"Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	// Pass to Service
	resp, err := h.service.CreateGroup(r.Context(), req)
	if err != nil {
		// In a real app, you'd check error types (e.g., duplicate email) for 409 Conflict
		http.Error(w, `{"status":"error","message":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
