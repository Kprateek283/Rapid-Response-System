package models

import (
	"time"

	"github.com/google/uuid"
)

// Incident status constants
const (
	IncidentStatusOpen        = "OPEN"
	IncidentStatusProcessing  = "PROCESSING"
	IncidentStatusDispatched  = "DISPATCHED"
	IncidentStatusArrived     = "ARRIVED"
	IncidentStatusAssessed    = "ASSESSED"
	IncidentStatusStabilizing = "STABILIZING"
	IncidentStatusControlled  = "CONTROLLED"
	IncidentStatusResolved    = "RESOLVED"
	IncidentStatusClosed      = "CLOSED"
)

// Dispatch assignment status constants
const (
	DispatchStatusPending          = "PENDING"
	DispatchStatusNotified         = "NOTIFIED"
	DispatchStatusAccepted         = "ACCEPTED"
	DispatchStatusDeclined         = "DECLINED"
	DispatchStatusInTransit        = "IN_TRANSIT"
	DispatchStatusArrived          = "ARRIVED"
	DispatchStatusOnSiteExpendable = "ON_SITE_EXPENDABLE"
	DispatchStatusReleased         = "RELEASED"
	DispatchStatusTimedOut         = "TIMED_OUT"
)

// Trigger source constants
const (
	TriggerManualPhone = "MANUAL_PHONE"
	TriggerBLEButton   = "BLE_BUTTON"
	TriggerVenueSensor = "VENUE_SENSOR"
	TriggerSMSWebhook  = "SMS_WEBHOOK"
)

// Incident represents a crisis incident in the system.
type Incident struct {
	ID                   string     `json:"id" db:"id"`
	HotelID              uuid.UUID  `json:"hotel_id" db:"hotel_id"`
	RoomID               uuid.UUID  `json:"room_id" db:"room_id"`
	GuestID              string     `json:"guest_id" db:"guest_id"`
	TriggerSource        string     `json:"trigger_source" db:"trigger_source"`
	GPSLat               *float64   `json:"gps_lat,omitempty" db:"gps_lat"`
	GPSLng               *float64   `json:"gps_lng,omitempty" db:"gps_lng"`
	Status               string     `json:"status" db:"status"`
	InitialSeverity      *int       `json:"initial_severity,omitempty" db:"initial_severity"`
	GroundTruthSeverity  *int       `json:"ground_truth_severity,omitempty" db:"ground_truth_severity"`
	FinalSeverity        *int       `json:"final_severity,omitempty" db:"final_severity"`
	AISummary            *string    `json:"ai_summary,omitempty" db:"ai_summary"`
	MediaVideoURL        *string    `json:"media_video_url,omitempty" db:"media_video_url"`
	MediaAudioStreamID   *string    `json:"media_audio_stream_id,omitempty" db:"media_audio_stream_id"`
	EscalationStatus     *string    `json:"escalation_status,omitempty" db:"escalation_status"`
	ResolutionProposedBy *string    `json:"resolution_proposed_by,omitempty" db:"resolution_proposed_by"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	ResolvedAt           *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ClosedAt             *time.Time `json:"closed_at,omitempty" db:"closed_at"`
}

// DispatchAssignment represents a staff-to-incident dispatch mapping.
type DispatchAssignment struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	IncidentID    string     `json:"incident_id" db:"incident_id"`
	StaffID       uuid.UUID  `json:"staff_id" db:"staff_id"`
	Status        string     `json:"status" db:"status"`
	AcceptedAt    *time.Time `json:"accepted_at,omitempty" db:"accepted_at"`
	ArrivedAt     *time.Time `json:"arrived_at,omitempty" db:"arrived_at"`
	ReleasedAt    *time.Time `json:"released_at,omitempty" db:"released_at"`
	DeclineReason *string    `json:"decline_reason,omitempty" db:"decline_reason"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// IncidentEvent represents an immutable audit log entry for an incident.
type IncidentEvent struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	IncidentID string                 `json:"incident_id" db:"incident_id"`
	EventType  string                 `json:"event_type" db:"event_type"`
	ActorID    *string                `json:"actor_id,omitempty" db:"actor_id"`
	ActorRole  *string                `json:"actor_role,omitempty" db:"actor_role"`
	Payload    map[string]interface{} `json:"payload" db:"payload"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// IncidentReport represents a final compiled report for a closed incident.
type IncidentReport struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	IncidentID string                 `json:"incident_id" db:"incident_id"`
	ReportJSON map[string]interface{} `json:"report_json" db:"report_json"`
	CompiledBy string                 `json:"compiled_by" db:"compiled_by"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}
