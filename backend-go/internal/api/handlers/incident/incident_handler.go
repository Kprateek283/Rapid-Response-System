package incident

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	incidentDto "github.com/google-hackathon/rapid-response/internal/api/dto/incident"
	"github.com/google-hackathon/rapid-response/internal/api/middleware"
	"github.com/google-hackathon/rapid-response/internal/models"
	incidentService "github.com/google-hackathon/rapid-response/internal/service/incident"
	"github.com/google/uuid"
)

// IncidentHandler handles all incident-related HTTP endpoints.
type IncidentHandler struct {
	service *incidentService.IncidentService
}

// NewIncidentHandler creates a new incident handler.
func NewIncidentHandler(service *incidentService.IncidentService) *IncidentHandler {
	return &IncidentHandler{service: service}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"status": "error", "message": message})
}

// TriggerInit handles POST /incident/trigger/init
func (h *IncidentHandler) TriggerInit(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok || claims.GuestID == "" {
		writeError(w, http.StatusUnauthorized, "Guest token required")
		return
	}

	var req incidentDto.TriggerInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.TriggerInit(r.Context(), claims.GuestID, claims.HotelID, claims.RoomID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// MediaReady handles POST /incident/:incident_id/media-ready
func (h *IncidentHandler) MediaReady(w http.ResponseWriter, r *http.Request) {
	incidentID := chi.URLParam(r, "incident_id")
	if incidentID == "" {
		writeError(w, http.StatusBadRequest, "incident_id required")
		return
	}

	resp, err := h.service.MediaReady(r.Context(), incidentID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// DispatchAccept handles POST /incident/:incident_id/dispatch/accept
func (h *IncidentHandler) DispatchAccept(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")
	staffID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid staff token")
		return
	}

	var req incidentDto.DispatchAcceptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.DispatchAccept(r.Context(), incidentID, staffID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// DispatchDecline handles POST /incident/:incident_id/dispatch/decline
func (h *IncidentHandler) DispatchDecline(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")
	staffID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid staff token")
		return
	}

	var req incidentDto.DispatchDeclineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.DispatchDecline(r.Context(), incidentID, staffID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// TransitPing handles POST /incident/:incident_id/transit/ping
func (h *IncidentHandler) TransitPing(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")
	staffID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid staff token")
		return
	}

	var req incidentDto.TransitPingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.TransitPing(r.Context(), incidentID, staffID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// StaffArrive handles POST /incident/:incident_id/staff/arrive
func (h *IncidentHandler) StaffArrive(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")
	staffID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid staff token")
		return
	}

	var req incidentDto.ArrivalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.StaffArrive(r.Context(), incidentID, staffID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GroundTruth handles POST /incident/:incident_id/staff/ground-truth
func (h *IncidentHandler) GroundTruth(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")
	staffID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid staff token")
		return
	}

	var req incidentDto.GroundTruthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.SubmitGroundTruth(r.Context(), incidentID, staffID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// RequestBackup handles POST /incident/:incident_id/action/backup
func (h *IncidentHandler) RequestBackup(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")
	staffID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid staff token")
		return
	}

	var req incidentDto.BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.RequestBackup(r.Context(), incidentID, staffID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// Escalate handles POST /incident/:incident_id/action/escalate
func (h *IncidentHandler) Escalate(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")

	var req incidentDto.EscalateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.InitiateEscalation(r.Context(), incidentID, claims.UserID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// EscalateAbort handles POST /incident/:incident_id/action/escalate/abort
func (h *IncidentHandler) EscalateAbort(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")

	var req incidentDto.EscalateAbortRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.AbortEscalation(r.Context(), incidentID, claims.UserID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ProposeResolution handles POST /incident/:incident_id/action/resolve/propose
func (h *IncidentHandler) ProposeResolution(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")

	var req incidentDto.ResolveProposeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resp, err := h.service.ProposeResolution(r.Context(), incidentID, claims.UserID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ConfirmResolution handles POST /incident/:incident_id/action/resolve/confirm
func (h *IncidentHandler) ConfirmResolution(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")

	resp, err := h.service.ConfirmResolution(r.Context(), incidentID, claims.UserID, claims.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
