package incident

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentRepository struct {
	db *pgxpool.Pool
}

func NewIncidentRepository(db *pgxpool.Pool) *IncidentRepository {
	return &IncidentRepository{db: db}
}

// generateIncidentID creates an ID in the format "inc_{random_hex(6)}"
func generateIncidentID() (string, error) {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "inc_" + hex.EncodeToString(bytes), nil
}

// CreateIncident inserts a new incident row and returns the generated ID.
func (r *IncidentRepository) CreateIncident(ctx context.Context, hotelID, roomID uuid.UUID, guestID, triggerSource string, lat, lng *float64) (string, error) {
	incidentID, err := generateIncidentID()
	if err != nil {
		return "", fmt.Errorf("failed to generate incident ID: %w", err)
	}

	query := `
		INSERT INTO incidents (id, hotel_id, room_id, guest_id, trigger_source, gps_lat, gps_lng, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'OPEN')
	`
	_, err = r.db.Exec(ctx, query, incidentID, hotelID, roomID, guestID, triggerSource, lat, lng)
	if err != nil {
		return "", fmt.Errorf("failed to create incident: %w", err)
	}

	return incidentID, nil
}

// GetIncidentByID retrieves an incident by ID.
func (r *IncidentRepository) GetIncidentByID(ctx context.Context, incidentID string) (*IncidentRow, error) {
	row := &IncidentRow{}
	query := `
		SELECT id, hotel_id, room_id, guest_id, trigger_source, gps_lat, gps_lng, status,
		       initial_severity, ground_truth_severity, final_severity, ai_summary,
		       media_video_url, escalation_status, resolution_proposed_by,
		       created_at, resolved_at, closed_at
		FROM incidents WHERE id = $1
	`
	err := r.db.QueryRow(ctx, query, incidentID).Scan(
		&row.ID, &row.HotelID, &row.RoomID, &row.GuestID, &row.TriggerSource,
		&row.GPSLat, &row.GPSLng, &row.Status,
		&row.InitialSeverity, &row.GroundTruthSeverity, &row.FinalSeverity, &row.AISummary,
		&row.MediaVideoURL, &row.EscalationStatus, &row.ResolutionProposedBy,
		&row.CreatedAt, &row.ResolvedAt, &row.ClosedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("incident not found: %w", err)
	}
	return row, nil
}

// IncidentRow is a scan-friendly struct for incident queries.
type IncidentRow struct {
	ID                   string
	HotelID              uuid.UUID
	RoomID               uuid.UUID
	GuestID              string
	TriggerSource        string
	GPSLat               *float64
	GPSLng               *float64
	Status               string
	InitialSeverity      *int
	GroundTruthSeverity  *int
	FinalSeverity        *int
	AISummary            *string
	MediaVideoURL        *string
	EscalationStatus     *string
	ResolutionProposedBy *string
	CreatedAt            time.Time
	ResolvedAt           *time.Time
	ClosedAt             *time.Time
}

// UpdateIncidentStatus updates the status of an incident.
func (r *IncidentRepository) UpdateIncidentStatus(ctx context.Context, incidentID, status string) error {
	query := `UPDATE incidents SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, incidentID)
	return err
}

// UpdateIncidentAI updates AI assessment fields.
func (r *IncidentRepository) UpdateIncidentAI(ctx context.Context, incidentID string, severity int, summary string) error {
	query := `UPDATE incidents SET initial_severity = $1, final_severity = $1, ai_summary = $2, status = 'PROCESSING' WHERE id = $3`
	_, err := r.db.Exec(ctx, query, severity, summary, incidentID)
	return err
}

// UpdateGroundTruth sets the ground truth severity.
func (r *IncidentRepository) UpdateGroundTruth(ctx context.Context, incidentID string, severity int) error {
	query := `UPDATE incidents SET ground_truth_severity = $1, status = 'ASSESSED' WHERE id = $2`
	_, err := r.db.Exec(ctx, query, severity, incidentID)
	return err
}

// UpdateFinalSeverity updates the final severity after re-evaluation.
func (r *IncidentRepository) UpdateFinalSeverity(ctx context.Context, incidentID string, severity int, summary string) error {
	query := `UPDATE incidents SET final_severity = $1, ai_summary = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, severity, summary, incidentID)
	return err
}

// SetResolutionProposed sets who proposed the resolution.
func (r *IncidentRepository) SetResolutionProposed(ctx context.Context, incidentID, proposedBy string) error {
	query := `UPDATE incidents SET resolution_proposed_by = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, proposedBy, incidentID)
	return err
}

// ResolveIncident sets the incident to RESOLVED.
func (r *IncidentRepository) ResolveIncident(ctx context.Context, incidentID string) error {
	query := `UPDATE incidents SET status = 'RESOLVED', resolved_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, incidentID)
	return err
}

// CloseIncident sets the incident to CLOSED.
func (r *IncidentRepository) CloseIncident(ctx context.Context, incidentID string) error {
	query := `UPDATE incidents SET status = 'CLOSED', closed_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, incidentID)
	return err
}

// SetEscalationStatus updates the escalation status.
func (r *IncidentRepository) SetEscalationStatus(ctx context.Context, incidentID, status string) error {
	query := `UPDATE incidents SET escalation_status = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, incidentID)
	return err
}

// --- Dispatch Assignments ---

// CreateDispatchAssignment inserts a new dispatch assignment.
func (r *IncidentRepository) CreateDispatchAssignment(ctx context.Context, incidentID string, staffID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	query := `
		INSERT INTO dispatch_assignments (incident_id, staff_id, status)
		VALUES ($1, $2, 'PENDING')
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query, incidentID, staffID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create dispatch assignment: %w", err)
	}
	return id, nil
}

// UpdateDispatchStatus updates a dispatch assignment's status.
func (r *IncidentRepository) UpdateDispatchStatus(ctx context.Context, incidentID string, staffID uuid.UUID, status string) error {
	query := `UPDATE dispatch_assignments SET status = $1 WHERE incident_id = $2 AND staff_id = $3`
	_, err := r.db.Exec(ctx, query, status, incidentID, staffID)
	return err
}

// UpdateDispatchAccepted sets accepted_at timestamp.
func (r *IncidentRepository) UpdateDispatchAccepted(ctx context.Context, incidentID string, staffID uuid.UUID) error {
	query := `UPDATE dispatch_assignments SET status = 'ACCEPTED', accepted_at = NOW() WHERE incident_id = $1 AND staff_id = $2`
	_, err := r.db.Exec(ctx, query, incidentID, staffID)
	return err
}

// UpdateDispatchArrived sets arrived_at timestamp.
func (r *IncidentRepository) UpdateDispatchArrived(ctx context.Context, incidentID string, staffID uuid.UUID) error {
	query := `UPDATE dispatch_assignments SET status = 'ARRIVED', arrived_at = NOW() WHERE incident_id = $1 AND staff_id = $2`
	_, err := r.db.Exec(ctx, query, incidentID, staffID)
	return err
}

// GetDispatchAssignments retrieves all dispatch assignments for an incident.
func (r *IncidentRepository) GetDispatchAssignments(ctx context.Context, incidentID string) ([]DispatchRow, error) {
	query := `
		SELECT da.id, da.incident_id, da.staff_id, da.status, s.name, s.role
		FROM dispatch_assignments da
		JOIN staff s ON da.staff_id = s.id
		WHERE da.incident_id = $1
	`
	rows, err := r.db.Query(ctx, query, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DispatchRow
	for rows.Next() {
		var d DispatchRow
		if err := rows.Scan(&d.ID, &d.IncidentID, &d.StaffID, &d.Status, &d.StaffName, &d.StaffRole); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}

// DispatchRow is a scan-friendly struct for dispatch queries.
type DispatchRow struct {
	ID         uuid.UUID
	IncidentID string
	StaffID    uuid.UUID
	Status     string
	StaffName  string
	StaffRole  string
}

// GetActiveDispatchForStaff finds active dispatch for a staff member on an incident.
func (r *IncidentRepository) GetActiveDispatchForStaff(ctx context.Context, incidentID string, staffID uuid.UUID) (*DispatchRow, error) {
	d := &DispatchRow{}
	query := `
		SELECT da.id, da.incident_id, da.staff_id, da.status, s.name, s.role
		FROM dispatch_assignments da
		JOIN staff s ON da.staff_id = s.id
		WHERE da.incident_id = $1 AND da.staff_id = $2 AND da.status NOT IN ('DECLINED', 'TIMED_OUT', 'RELEASED')
	`
	err := r.db.QueryRow(ctx, query, incidentID, staffID).Scan(&d.ID, &d.IncidentID, &d.StaffID, &d.Status, &d.StaffName, &d.StaffRole)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// --- Incident Events ---

// LogEvent inserts an audit log entry.
func (r *IncidentRepository) LogEvent(ctx context.Context, incidentID, eventType string, actorID, actorRole *string, payload map[string]interface{}) error {
	payloadJSON, _ := json.Marshal(payload)
	query := `
		INSERT INTO incident_events (incident_id, event_type, actor_id, actor_role, payload)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query, incidentID, eventType, actorID, actorRole, payloadJSON)
	return err
}

// GetEventsByIncident retrieves all events for an incident.
func (r *IncidentRepository) GetEventsByIncident(ctx context.Context, incidentID string) ([]EventRow, error) {
	query := `
		SELECT id, incident_id, event_type, actor_id, actor_role, payload, created_at
		FROM incident_events WHERE incident_id = $1 ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []EventRow
	for rows.Next() {
		var e EventRow
		if err := rows.Scan(&e.ID, &e.IncidentID, &e.EventType, &e.ActorID, &e.ActorRole, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

// EventRow is a scan-friendly struct for event queries.
type EventRow struct {
	ID         uuid.UUID
	IncidentID string
	EventType  string
	ActorID    *string
	ActorRole  *string
	Payload    json.RawMessage
	CreatedAt  time.Time
}

// --- Incident Reports ---

// CreateReport inserts a final report.
func (r *IncidentRepository) CreateReport(ctx context.Context, incidentID string, reportJSON map[string]interface{}, compiledBy string) error {
	rJSON, _ := json.Marshal(reportJSON)
	query := `
		INSERT INTO incident_reports (incident_id, report_json, compiled_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (incident_id) DO UPDATE SET report_json = $2, compiled_by = $3
	`
	_, err := r.db.Exec(ctx, query, incidentID, rJSON, compiledBy)
	return err
}

// --- Spatial Queries ---

// FindNearestStaff finds the nearest available staff by role using PostGIS.
func (r *IncidentRepository) FindNearestStaff(ctx context.Context, hotelID uuid.UUID, role string, skillKey string, skillMin float64, limit int) ([]StaffSpatialRow, error) {
	query := `
		SELECT s.id, s.name, s.role, s.skill_profile,
		       ST_Distance(h.coordinates::geography, ST_SetSRID(ST_MakePoint(0, 0), 4326)::geography) as distance
		FROM staff s
		JOIN hotels h ON s.hotel_id = h.id
		WHERE s.hotel_id = $1
		  AND s.role = $2
		  AND (s.skill_profile->>$3)::float >= $4
		ORDER BY s.id
		LIMIT $5
	`
	rows, err := r.db.Query(ctx, query, hotelID, role, skillKey, skillMin, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []StaffSpatialRow
	for rows.Next() {
		var s StaffSpatialRow
		if err := rows.Scan(&s.ID, &s.Name, &s.Role, &s.SkillProfile, &s.Distance); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

// StaffSpatialRow is a scan-friendly struct for spatial queries.
type StaffSpatialRow struct {
	ID           uuid.UUID
	Name         string
	Role         string
	SkillProfile json.RawMessage
	Distance     float64
}

// GetRoomCoordinates retrieves the hotel coordinates for a room (for geofence verification).
func (r *IncidentRepository) GetRoomCoordinates(ctx context.Context, roomID uuid.UUID) (float64, float64, error) {
	var lat, lng float64
	query := `
		SELECT ST_Y(h.coordinates::geometry) as lat, ST_X(h.coordinates::geometry) as lng
		FROM rooms r
		JOIN hotels h ON r.hotel_id = h.id
		WHERE r.id = $1
	`
	err := r.db.QueryRow(ctx, query, roomID).Scan(&lat, &lng)
	if err != nil {
		return 0, 0, fmt.Errorf("room coordinates not found: %w", err)
	}
	return lat, lng, nil
}

// CountActiveDispatches counts non-terminal dispatch assignments for an incident.
func (r *IncidentRepository) CountActiveDispatches(ctx context.Context, incidentID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM dispatch_assignments WHERE incident_id = $1 AND status NOT IN ('DECLINED', 'TIMED_OUT', 'RELEASED')`
	err := r.db.QueryRow(ctx, query, incidentID).Scan(&count)
	return count, err
}

// GetStaffHotelID retrieves the hotel_id for a staff member.
func (r *IncidentRepository) GetStaffHotelID(ctx context.Context, staffID uuid.UUID) (uuid.UUID, error) {
	var hotelID uuid.UUID
	query := `SELECT hotel_id FROM staff WHERE id = $1`
	err := r.db.QueryRow(ctx, query, staffID).Scan(&hotelID)
	return hotelID, err
}

// GetRoomNumberByID retrieves the room number for display purposes.
func (r *IncidentRepository) GetRoomNumberByID(ctx context.Context, roomID uuid.UUID) (string, error) {
	var roomNumber string
	query := `SELECT room_number FROM rooms WHERE id = $1`
	err := r.db.QueryRow(ctx, query, roomID).Scan(&roomNumber)
	return roomNumber, err
}
