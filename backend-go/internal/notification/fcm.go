package notification

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DispatchPayload contains the data sent to staff when dispatched.
type DispatchPayload struct {
	IncidentID string  `json:"incident_id"`
	RoomNumber string  `json:"room_number"`
	Severity   int     `json:"severity"`
	Role       string  `json:"role"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Message    string  `json:"message"`
}

// Notifier defines the interface for sending push notifications.
type Notifier interface {
	SendDispatchAlarm(ctx context.Context, staffID string, payload DispatchPayload) error
	SendStandDown(ctx context.Context, staffID string, reason string) error
	SendEscalationAlert(ctx context.Context, managerID string, incidentID string, message string) error
}

// MockNotifier implements Notifier by logging and saving to DB.
// In production, this would use Firebase Admin SDK for FCM.
type MockNotifier struct {
	db *pgxpool.Pool
}

// NewMockNotifier creates a new mock notifier.
func NewMockNotifier(db *pgxpool.Pool) *MockNotifier {
	return &MockNotifier{db: db}
}

func (n *MockNotifier) SendDispatchAlarm(ctx context.Context, staffID string, payload DispatchPayload) error {
	log.Printf("[FCM MOCK] 📲 Dispatch alarm sent to staff %s: incident=%s severity=%d role=%s",
		staffID, payload.IncidentID, payload.Severity, payload.Role)

	// Log as incident event
	payloadJSON, _ := json.Marshal(payload)
	_, err := n.db.Exec(ctx, `
		INSERT INTO incident_events (incident_id, event_type, actor_id, actor_role, payload)
		VALUES ($1, 'FCM_DISPATCH_ALARM', $2, 'SYSTEM', $3)
	`, payload.IncidentID, staffID, payloadJSON)

	return err
}

func (n *MockNotifier) SendStandDown(ctx context.Context, staffID string, reason string) error {
	log.Printf("[FCM MOCK] 🛑 Stand-down sent to staff %s: %s", staffID, reason)

	// ASSUMPTION: We don't have the incident_id here, just log it
	return nil
}

func (n *MockNotifier) SendEscalationAlert(ctx context.Context, managerID string, incidentID string, message string) error {
	log.Printf("[FCM MOCK] 🚨 Escalation alert sent to manager %s for incident %s: %s",
		managerID, incidentID, message)

	payloadJSON, _ := json.Marshal(map[string]string{"message": message, "manager_id": managerID})
	_, err := n.db.Exec(ctx, `
		INSERT INTO incident_events (incident_id, event_type, actor_id, actor_role, payload)
		VALUES ($1, 'FCM_ESCALATION_ALERT', $2, 'SYSTEM', $3)
	`, incidentID, managerID, payloadJSON)

	return err
}
