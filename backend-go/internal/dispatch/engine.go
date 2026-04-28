package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/google-hackathon/rapid-response/internal/mock"
	"github.com/google-hackathon/rapid-response/internal/notification"
	incidentRepo "github.com/google-hackathon/rapid-response/internal/repository/incident"
	"github.com/google-hackathon/rapid-response/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Engine handles dispatch logic: priority queue, spatial resolution, cascade routing, steal logic.
type Engine struct {
	db       *pgxpool.Pool
	redis    *redis.Client
	hub      *ws.Hub
	notifier notification.Notifier
	repo     *incidentRepo.IncidentRepository
}

// NewEngine creates a new Dispatch Engine.
func NewEngine(db *pgxpool.Pool, redisClient *redis.Client, hub *ws.Hub, notifier notification.Notifier, repo *incidentRepo.IncidentRepository) *Engine {
	return &Engine{
		db:       db,
		redis:    redisClient,
		hub:      hub,
		notifier: notifier,
		repo:     repo,
	}
}

// EnqueueIncident adds an incident to the hotel's dispatch priority queue.
// Score = severity + (1.0 / unixTimestamp) — higher severity wins, ties broken by recency.
func (e *Engine) EnqueueIncident(ctx context.Context, hotelID, incidentID string, severity int) {
	score := float64(severity) + (1.0 / float64(time.Now().Unix()))
	key := "hotel:" + hotelID + ":dispatch_queue"
	e.redis.ZAdd(ctx, key, redis.Z{Score: score, Member: incidentID})
	log.Printf("[Dispatch] Enqueued incident %s with score %.6f", incidentID, score)
}

// DequeueHighest pops the highest-priority incident from the queue.
func (e *Engine) DequeueHighest(ctx context.Context, hotelID string) (string, error) {
	key := "hotel:" + hotelID + ":dispatch_queue"
	results, err := e.redis.ZRevRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: "-inf", Max: "+inf", Count: 1,
	}).Result()
	if err != nil || len(results) == 0 {
		return "", fmt.Errorf("no incidents in queue")
	}
	// Remove from queue
	e.redis.ZRem(ctx, key, results[0])
	return results[0], nil
}

// RunDispatch executes the full dispatch flow for an incident after AI assessment.
func (e *Engine) RunDispatch(ctx context.Context, incidentID string, assessment mock.MockAssessmentResult) {
	log.Printf("[Dispatch] Running dispatch for incident %s (severity=%d)", incidentID, assessment.FinalSeverity)

	// Get incident details
	incident, err := e.repo.GetIncidentByID(ctx, incidentID)
	if err != nil {
		log.Printf("[Dispatch] Error getting incident: %v", err)
		return
	}

	// Update incident with AI results
	if err := e.repo.UpdateIncidentAI(ctx, incidentID, assessment.FinalSeverity, assessment.AISummary); err != nil {
		log.Printf("[Dispatch] Error updating AI results: %v", err)
	}

	// Enqueue in priority queue
	e.EnqueueIncident(ctx, incident.HotelID.String(), incidentID, assessment.FinalSeverity)

	// Get room number for notifications
	roomNumber, _ := e.repo.GetRoomNumberByID(ctx, incident.RoomID)

	// Dispatch staff for each required role
	for _, req := range assessment.RequiredStaff {
		e.dispatchRole(ctx, incident, incidentID, req, roomNumber, assessment.FinalSeverity)
	}

	// Update incident status to DISPATCHED
	if err := e.repo.UpdateIncidentStatus(ctx, incidentID, "DISPATCHED"); err != nil {
		log.Printf("[Dispatch] Error updating status: %v", err)
	}

	// Broadcast dispatch event via WS
	e.hub.BroadcastEvent(incidentID, ws.WSEvent{
		Event: "DISPATCH_INITIATED",
		Payload: map[string]interface{}{
			"severity":       assessment.FinalSeverity,
			"ai_summary":     assessment.AISummary,
			"required_staff": assessment.RequiredStaff,
		},
	})
}

// dispatchRole finds and dispatches staff for a specific role requirement.
func (e *Engine) dispatchRole(ctx context.Context, incident *incidentRepo.IncidentRow, incidentID string, req mock.RoleRequirement, roomNumber string, severity int) {
	// Find nearest available staff using PostGIS
	for skillKey, skillMin := range req.SkillThresholds {
		candidates, err := e.repo.FindNearestStaff(ctx, incident.HotelID, req.Role, skillKey, skillMin, req.MinCount*3)
		if err != nil {
			log.Printf("[Dispatch] Error finding staff for role %s: %v", req.Role, err)
			continue
		}

		dispatched := 0
		for _, candidate := range candidates {
			if dispatched >= req.MinCount {
				break
			}

			// Check Redis availability
			statusKey := fmt.Sprintf("staff:%s:%s:status", incident.HotelID.String(), candidate.ID.String())
			status, err := e.redis.Get(ctx, statusKey).Result()
			if err != nil || status != "AVAILABLE" {
				continue
			}

			// Create dispatch assignment
			_, err = e.repo.CreateDispatchAssignment(ctx, incidentID, candidate.ID)
			if err != nil {
				log.Printf("[Dispatch] Error creating assignment: %v", err)
				continue
			}

			// Set status to NOTIFIED
			e.repo.UpdateDispatchStatus(ctx, incidentID, candidate.ID, "NOTIFIED")

			// Send FCM notification (mocked)
			e.notifier.SendDispatchAlarm(ctx, candidate.ID.String(), notification.DispatchPayload{
				IncidentID: incidentID,
				RoomNumber: roomNumber,
				Severity:   severity,
				Role:       req.Role,
				Message:    fmt.Sprintf("Emergency in Room %s. Severity: %d. Proceed immediately.", roomNumber, severity),
			})

			// Log event
			actorID := "SYSTEM"
			actorRole := "DISPATCH_ENGINE"
			e.repo.LogEvent(ctx, incidentID, "STAFF_DISPATCHED", &actorID, &actorRole, map[string]interface{}{
				"staff_id":   candidate.ID.String(),
				"staff_name": candidate.Name,
				"role":       req.Role,
			})

			// Start cascade timeout (15s to accept)
			go e.cascadeTimeout(ctx, incidentID, candidate.ID, incident.HotelID, req, roomNumber, severity)

			dispatched++
			log.Printf("[Dispatch] Dispatched %s (%s) for incident %s", candidate.Name, req.Role, incidentID)
		}

		if dispatched < req.MinCount {
			log.Printf("[Dispatch] WARNING: Only dispatched %d/%d for role %s. Attempting steal logic.", dispatched, req.MinCount, req.Role)
			e.attemptSteal(ctx, incident.HotelID, incidentID, req, severity, req.MinCount-dispatched)
		}
	}
}

// cascadeTimeout waits 15s for staff to accept, then reassigns if not accepted.
func (e *Engine) cascadeTimeout(ctx context.Context, incidentID string, staffID, hotelID uuid.UUID, req mock.RoleRequirement, roomNumber string, severity int) {
	time.Sleep(15 * time.Second)

	// Check if staff accepted
	dispatch, err := e.repo.GetActiveDispatchForStaff(ctx, incidentID, staffID)
	if err != nil {
		return // Already handled
	}

	if dispatch.Status == "NOTIFIED" || dispatch.Status == "PENDING" {
		// Timed out — mark and reassign
		log.Printf("[Dispatch] Staff %s timed out for incident %s", staffID.String(), incidentID)
		e.repo.UpdateDispatchStatus(ctx, incidentID, staffID, "TIMED_OUT")

		// Reset staff availability
		statusKey := fmt.Sprintf("staff:%s:%s:status", hotelID.String(), staffID.String())
		e.redis.Set(ctx, statusKey, "AVAILABLE", 0)

		// Log event
		actorID := "SYSTEM"
		actorRole := "DISPATCH_ENGINE"
		e.repo.LogEvent(ctx, incidentID, "DISPATCH_TIMED_OUT", &actorID, &actorRole, map[string]interface{}{
			"staff_id": staffID.String(),
		})

		// Try next candidate
		incident, err := e.repo.GetIncidentByID(ctx, incidentID)
		if err != nil {
			return
		}
		e.dispatchRole(ctx, incident, incidentID, req, roomNumber, severity)
	}
}

// attemptSteal tries to steal staff from lower-priority incidents.
func (e *Engine) attemptSteal(ctx context.Context, hotelID uuid.UUID, incidentID string, req mock.RoleRequirement, severity int, deficit int) {
	key := "hotel:" + hotelID.String() + ":dispatch_queue"

	// Find incidents with lower priority
	lowerIncidents, err := e.redis.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: "-inf", Max: fmt.Sprintf("%f", float64(severity)-0.001),
	}).Result()
	if err != nil || len(lowerIncidents) == 0 {
		log.Printf("[Dispatch] No lower-priority incidents to steal from")
		return
	}

	for _, lowerIncID := range lowerIncidents {
		if deficit <= 0 {
			break
		}

		dispatches, err := e.repo.GetDispatchAssignments(ctx, lowerIncID)
		if err != nil {
			continue
		}

		for _, d := range dispatches {
			if deficit <= 0 {
				break
			}
			if d.StaffRole == req.Role && d.Status == "ON_SITE_EXPENDABLE" {
				// Steal this staff member
				e.repo.UpdateDispatchStatus(ctx, lowerIncID, d.StaffID, "RELEASED")

				// Create new assignment for higher-priority incident
				e.repo.CreateDispatchAssignment(ctx, incidentID, d.StaffID)
				e.repo.UpdateDispatchStatus(ctx, incidentID, d.StaffID, "NOTIFIED")

				// Send stand-down for old + dispatch for new
				e.notifier.SendStandDown(ctx, d.StaffID.String(), "Reassigned to higher-priority incident")
				e.notifier.SendDispatchAlarm(ctx, d.StaffID.String(), notification.DispatchPayload{
					IncidentID: incidentID,
					Severity:   severity,
					Role:       req.Role,
					Message:    "Reassigned: Higher-priority emergency.",
				})

				actorID := "SYSTEM"
				actorRole := "DISPATCH_ENGINE"
				e.repo.LogEvent(ctx, incidentID, "STAFF_STOLEN", &actorID, &actorRole, map[string]interface{}{
					"staff_id":      d.StaffID.String(),
					"from_incident": lowerIncID,
				})

				deficit--
				log.Printf("[Dispatch] Stole staff %s from incident %s", d.StaffID.String(), lowerIncID)
			}
		}
	}
}

// RunDeltaEngine re-evaluates dispatch requirements after ground truth.
func (e *Engine) RunDeltaEngine(ctx context.Context, incidentID string, newAssessment mock.MockAssessmentResult) {
	log.Printf("[Delta] Running delta engine for incident %s", incidentID)

	incident, err := e.repo.GetIncidentByID(ctx, incidentID)
	if err != nil {
		log.Printf("[Delta] Error: %v", err)
		return
	}

	// Update final severity
	e.repo.UpdateFinalSeverity(ctx, incidentID, newAssessment.FinalSeverity, newAssessment.AISummary)

	// Get current active dispatches
	dispatches, err := e.repo.GetDispatchAssignments(ctx, incidentID)
	if err != nil {
		return
	}

	// Count active (non-terminal) dispatches by role
	activeByRole := make(map[string]int)
	for _, d := range dispatches {
		if d.Status != "DECLINED" && d.Status != "TIMED_OUT" && d.Status != "RELEASED" {
			activeByRole[d.StaffRole]++
		}
	}

	// Compare with new requirements
	for _, req := range newAssessment.RequiredStaff {
		active := activeByRole[req.Role]
		if active > req.MinCount {
			// OVER-DISPATCHED — mark extras as expendable
			excess := active - req.MinCount
			log.Printf("[Delta] Over-dispatched %s by %d, sending STAND_DOWN", req.Role, excess)
			released := 0
			for _, d := range dispatches {
				if released >= excess {
					break
				}
				if d.StaffRole == req.Role && (d.Status == "ARRIVED" || d.Status == "ACCEPTED") {
					e.repo.UpdateDispatchStatus(ctx, incidentID, d.StaffID, "ON_SITE_EXPENDABLE")
					e.hub.BroadcastEvent(incidentID, ws.WSEvent{
						Event: "STAND_DOWN",
						Payload: map[string]interface{}{
							"staff_id": d.StaffID.String(),
							"role":     d.StaffRole,
							"message":  "You are no longer needed. Stand by as backup.",
						},
					})
					released++
				}
			}
		} else if active < req.MinCount {
			// UNDER-DISPATCHED — dispatch more
			deficit := req.MinCount - active
			log.Printf("[Delta] Under-dispatched %s by %d, dispatching more", req.Role, deficit)
			roomNumber, _ := e.repo.GetRoomNumberByID(ctx, incident.RoomID)
			e.dispatchRole(ctx, incident, incidentID, req, roomNumber, newAssessment.FinalSeverity)
			_ = deficit
		}
	}
}

// CheckTransitIdle checks if a staff member has stopped moving during transit.
// Returns true if staff is idle (< 2m movement over 3 pings).
func (e *Engine) CheckTransitIdle(ctx context.Context, incidentID, staffID string, lat, lng float64) bool {
	key := fmt.Sprintf("incident:%s:transit:%s", incidentID, staffID)

	// Store the ping as JSON
	pingData, _ := json.Marshal(map[string]float64{"lat": lat, "lng": lng})
	e.redis.LPush(ctx, key, string(pingData))
	e.redis.LTrim(ctx, key, 0, 2) // Keep only last 3
	e.redis.Expire(ctx, key, 5*time.Minute)

	// Get all pings
	pings, err := e.redis.LRange(ctx, key, 0, 2).Result()
	if err != nil || len(pings) < 3 {
		return false // Not enough data
	}

	// Parse first and last ping
	var first, last map[string]float64
	json.Unmarshal([]byte(pings[0]), &first)
	json.Unmarshal([]byte(pings[2]), &last)

	// Calculate distance in meters (Haversine approximation)
	distance := haversineDistance(first["lat"], first["lng"], last["lat"], last["lng"])

	// ASSUMPTION: < 2 meters over 3 pings = idle
	return distance < 2.0
}

// haversineDistance calculates distance between two GPS points in meters.
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000 // Earth radius in meters
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// VerifyArrival checks if staff GPS is within 10m of room coordinates.
func (e *Engine) VerifyArrival(ctx context.Context, roomID uuid.UUID, staffLat, staffLng float64) (bool, error) {
	roomLat, roomLng, err := e.repo.GetRoomCoordinates(ctx, roomID)
	if err != nil {
		return false, err
	}

	distance := haversineDistance(roomLat, roomLng, staffLat, staffLng)
	// ASSUMPTION: 10 meter geofence for arrival verification
	return distance < 10.0, nil
}

// ReleaseAllStaff releases all staff assigned to an incident back to AVAILABLE.
func (e *Engine) ReleaseAllStaff(ctx context.Context, incidentID string) {
	dispatches, err := e.repo.GetDispatchAssignments(ctx, incidentID)
	if err != nil {
		return
	}

	incident, err := e.repo.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return
	}

	for _, d := range dispatches {
		if d.Status != "DECLINED" && d.Status != "TIMED_OUT" && d.Status != "RELEASED" {
			e.repo.UpdateDispatchStatus(ctx, incidentID, d.StaffID, "RELEASED")
			statusKey := fmt.Sprintf("staff:%s:%s:status", incident.HotelID.String(), d.StaffID.String())
			e.redis.Set(ctx, statusKey, "AVAILABLE", 0)
		}
	}
	log.Printf("[Dispatch] Released all staff for incident %s", incidentID)
}

// CleanupRedisKeys removes all Redis keys associated with an incident.
func (e *Engine) CleanupRedisKeys(ctx context.Context, incidentID, hotelID string) {
	// Remove from dispatch queue
	key := "hotel:" + hotelID + ":dispatch_queue"
	e.redis.ZRem(ctx, key, incidentID)

	// Remove escalation timer
	e.redis.Del(ctx, "incident:"+incidentID+":escalation_timer")

	// ASSUMPTION: Transit keys expire naturally (5min TTL)
	log.Printf("[Dispatch] Cleaned up Redis keys for incident %s", incidentID)
}
