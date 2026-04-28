package incident

import (
	"github.com/google/uuid"
)

// --- Incident Trigger DTOs ---

type TriggerInitRequest struct {
	RoomID        uuid.UUID `json:"room_id"`
	TriggerSource string    `json:"trigger_source"`
	GPSLat        float64   `json:"gps_lat"`
	GPSLng        float64   `json:"gps_lng"`
}

type MediaUploadInfo struct {
	UploadURL        string `json:"upload_url"`
	Method           string `json:"method"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type TriggerInitResponse struct {
	Status string          `json:"status"`
	Data   TriggerInitData `json:"data"`
}

type TriggerInitData struct {
	IncidentID  string          `json:"incident_id"`
	WSURL       string          `json:"ws_url"`
	MediaUpload MediaUploadInfo `json:"media_upload"`
}

// --- Media Ready ---

type MediaReadyResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// --- Dispatch Accept/Decline ---

type DispatchAcceptRequest struct {
	BatteryLevel int     `json:"battery_level"`
	CurrentLat   float64 `json:"current_lat"`
	CurrentLng   float64 `json:"current_lng"`
}

type DispatchAcceptResponse struct {
	Status string             `json:"status"`
	Data   DispatchAcceptData `json:"data"`
}

type DispatchAcceptData struct {
	DispatchStatus string `json:"dispatch_status"`
	ChatRoomWSS    string `json:"chat_room_wss"`
	Message        string `json:"message"`
}

type DispatchDeclineRequest struct {
	Reason string `json:"reason"` // BATTERY_CRITICAL, MANUAL_DECLINE, ENGAGED_ELSEWHERE
}

type DispatchDeclineResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// --- Transit Ping ---

type TransitPingRequest struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Timestamp string  `json:"timestamp"`
}

type TransitPingResponse struct {
	Status string `json:"status"`
}

// --- Arrival ---

type ArrivalRequest struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Timestamp string  `json:"timestamp"`
}

type ArrivalResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// --- Ground Truth ---

type GroundTruthRequest struct {
	ManualSeverity int      `json:"manual_severity"`
	ScoutNotes     string   `json:"scout_notes"`
	MediaURLs      []string `json:"media_urls"`
}

type GroundTruthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// --- Branching Actions ---

type BackupRequest struct {
	AdditionalRolesNeeded map[string]int `json:"additional_roles_needed"`
	ReasonAudioURL        string         `json:"reason_audio_url"`
}

type BackupResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type EscalateRequest struct {
	TargetAuthority string `json:"target_authority"` // POLICE, FIRE, AMBULANCE
	Reason          string `json:"reason"`
}

type EscalateResponse struct {
	Status       string `json:"status"`
	TimerSeconds int    `json:"timer_seconds"`
	Message      string `json:"message"`
}

type EscalateAbortRequest struct {
	Reason string `json:"reason"`
}

type EscalateAbortResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ResolveProposeRequest struct {
	ProposedByRole  string `json:"proposed_by_role"`
	ResolutionNotes string `json:"resolution_notes"`
}

type ResolveProposeResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ResolveConfirmResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
