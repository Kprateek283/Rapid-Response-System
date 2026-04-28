package mock

import (
	"time"

	"github.com/google-hackathon/rapid-response/internal/models"
	incidentRepo "github.com/google-hackathon/rapid-response/internal/repository/incident"
	"github.com/google/uuid"
)

// EventModel is a simplified event struct for report compilation.
type EventModel struct {
	ID         uuid.UUID
	IncidentID string
	EventType  string
	ActorID    *string
	ActorRole  *string
	CreatedAt  time.Time
}

// IncidentToModel converts an IncidentRow to a models.Incident for mock functions.
func IncidentToModel(row *incidentRepo.IncidentRow) models.Incident {
	return models.Incident{
		ID:            row.ID,
		HotelID:       row.HotelID,
		RoomID:        row.RoomID,
		GuestID:       row.GuestID,
		TriggerSource: row.TriggerSource,
		GPSLat:        row.GPSLat,
		GPSLng:        row.GPSLng,
		Status:        row.Status,
		FinalSeverity: row.FinalSeverity,
		AISummary:     row.AISummary,
	}
}

// MockCompileReportFromRows compiles a report using EventModel rows (avoids import cycle).
func MockCompileReportFromRows(incident models.Incident, events []EventModel) map[string]interface{} {
	timeline := make([]map[string]interface{}, len(events))
	for i, e := range events {
		timeline[i] = map[string]interface{}{
			"event_type": e.EventType,
			"actor_id":   e.ActorID,
			"actor_role": e.ActorRole,
			"created_at": e.CreatedAt,
		}
	}

	severity := 0
	if incident.FinalSeverity != nil {
		severity = *incident.FinalSeverity
	}

	return map[string]interface{}{
		"incident_id":    incident.ID,
		"hotel_id":       incident.HotelID,
		"room_id":        incident.RoomID,
		"guest_id":       incident.GuestID,
		"trigger_source": incident.TriggerSource,
		"timeline":       timeline,
		"final_severity": severity,
		"ai_summary":     incident.AISummary,
		"resolution":     "Incident resolved via 2-party handshake",
		"compiled_at":    time.Now().UTC(),
		"compiled_by":    "MOCK_AI_AGENT",
	}
}
