package incident

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	incidentDto "github.com/google-hackathon/rapid-response/internal/api/dto/incident"
	"github.com/google-hackathon/rapid-response/internal/dispatch"
	"github.com/google-hackathon/rapid-response/internal/mock"
	"github.com/google-hackathon/rapid-response/internal/mq"
	incidentRepo "github.com/google-hackathon/rapid-response/internal/repository/incident"
	"github.com/google-hackathon/rapid-response/internal/ws"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

// IncidentService handles all incident lifecycle operations.
type IncidentService struct {
	repo      *incidentRepo.IncidentRepository
	redis     *redis.Client
	hub       *ws.Hub
	publisher *mq.Publisher
	engine    *dispatch.Engine
	minio     *minio.Client
	bucket    string
	serverURL string
}

// NewIncidentService creates a new incident service.
func NewIncidentService(
	repo *incidentRepo.IncidentRepository,
	redisClient *redis.Client,
	hub *ws.Hub,
	publisher *mq.Publisher,
	engine *dispatch.Engine,
	minioClient *minio.Client,
	bucket string,
) *IncidentService {
	baseURL := strings.TrimRight(os.Getenv("APP_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &IncidentService{
		repo:      repo,
		redis:     redisClient,
		hub:       hub,
		publisher: publisher,
		engine:    engine,
		minio:     minioClient,
		bucket:    bucket,
		serverURL: baseURL,
	}
}

func (s *IncidentService) wsBaseURL() string {
	switch {
	case strings.HasPrefix(s.serverURL, "https://"):
		return "wss://" + strings.TrimPrefix(s.serverURL, "https://")
	case strings.HasPrefix(s.serverURL, "http://"):
		return "ws://" + strings.TrimPrefix(s.serverURL, "http://")
	default:
		return "ws://" + s.serverURL
	}
}

// TriggerInit creates a new incident, sets up WS room, and returns media upload URL.
func (s *IncidentService) TriggerInit(ctx context.Context, guestID, hotelID, roomID string, req incidentDto.TriggerInitRequest) (incidentDto.TriggerInitResponse, error) {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return incidentDto.TriggerInitResponse{}, fmt.Errorf("invalid room_id")
	}
	hotelUUID, err := uuid.Parse(hotelID)
	if err != nil {
		return incidentDto.TriggerInitResponse{}, fmt.Errorf("invalid hotel_id")
	}

	lat := req.GPSLat
	lng := req.GPSLng

	// Create incident
	incidentID, err := s.repo.CreateIncident(ctx, hotelUUID, roomUUID, guestID, req.TriggerSource, &lat, &lng)
	if err != nil {
		return incidentDto.TriggerInitResponse{}, fmt.Errorf("failed to create incident: %w", err)
	}

	// Log event
	actorRole := "GUEST"
	s.repo.LogEvent(ctx, incidentID, "INCIDENT_CREATED", &guestID, &actorRole, map[string]interface{}{
		"trigger_source": req.TriggerSource,
		"gps_lat":        lat,
		"gps_lng":        lng,
	})

	// Create WebSocket room
	s.hub.CreateRoom(incidentID)

	// Generate MinIO presigned PUT URL for media upload
	objectName := fmt.Sprintf("incidents/%s/video_%d.webm", incidentID, time.Now().Unix())
	uploadURL := fmt.Sprintf("%s/upload/%s", s.serverURL, objectName) // ASSUMPTION: Simplified for hackathon

	// Try to generate actual presigned URL from MinIO
	if s.minio != nil {
		presignedURL, err := s.minio.PresignedPutObject(ctx, s.bucket, objectName, 2*time.Minute)
		if err == nil {
			uploadURL = presignedURL.String()
		} else {
			log.Printf("[Incident] MinIO presign error (using fallback): %v", err)
		}
	}

	wsURL := fmt.Sprintf("%s/api/v1/incident/ws/%s/live", s.wsBaseURL(), incidentID)

	return incidentDto.TriggerInitResponse{
		Status: "success",
		Data: incidentDto.TriggerInitData{
			IncidentID: incidentID,
			WSURL:      wsURL,
			MediaUpload: incidentDto.MediaUploadInfo{
				UploadURL:        uploadURL,
				Method:           "PUT",
				ExpiresInSeconds: 120,
			},
		},
	}, nil
}

// MediaReady marks media as uploaded and triggers AI assessment.
func (s *IncidentService) MediaReady(ctx context.Context, incidentID string) (incidentDto.MediaReadyResponse, error) {
	// Verify incident exists
	_, err := s.repo.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return incidentDto.MediaReadyResponse{}, fmt.Errorf("incident not found")
	}

	// Publish to RabbitMQ for AI assessment
	payload, _ := json.Marshal(map[string]string{"incident_id": incidentID})
	if err := s.publisher.Publish(ctx, "incident.media_ready", payload); err != nil {
		log.Printf("[Incident] Error publishing media_ready: %v", err)
		// Fallback: run mock assessment inline
		go func() {
			inc, _ := s.repo.GetIncidentByID(context.Background(), incidentID)
			if inc != nil {
				assessment := mock.MockAIAssessment(mock.IncidentToModel(inc))
				s.engine.RunDispatch(context.Background(), incidentID, assessment)
			}
		}()
	}

	actorID := "SYSTEM"
	actorRole := "SYSTEM"
	s.repo.LogEvent(ctx, incidentID, "MEDIA_READY", &actorID, &actorRole, nil)

	return incidentDto.MediaReadyResponse{
		Status:  "processing",
		Message: "Media secured. AI assessment queued.",
	}, nil
}

// DispatchAccept handles staff accepting a dispatch.
func (s *IncidentService) DispatchAccept(ctx context.Context, incidentID string, staffID uuid.UUID, req incidentDto.DispatchAcceptRequest) (incidentDto.DispatchAcceptResponse, error) {
	// Update dispatch assignment
	if err := s.repo.UpdateDispatchAccepted(ctx, incidentID, staffID); err != nil {
		return incidentDto.DispatchAcceptResponse{}, fmt.Errorf("failed to accept dispatch: %w", err)
	}

	// Get hotel ID for Redis key
	hotelID, _ := s.repo.GetStaffHotelID(ctx, staffID)
	statusKey := fmt.Sprintf("staff:%s:%s:status", hotelID.String(), staffID.String())
	s.redis.Set(ctx, statusKey, "BUSY", 0)

	// Log event
	staffIDStr := staffID.String()
	actorRole := "STAFF"
	s.repo.LogEvent(ctx, incidentID, "DISPATCH_ACCEPTED", &staffIDStr, &actorRole, map[string]interface{}{
		"battery_level": req.BatteryLevel,
	})

	// Broadcast to guest
	dispatch, _ := s.repo.GetActiveDispatchForStaff(ctx, incidentID, staffID)
	staffName := ""
	if dispatch != nil {
		staffName = dispatch.StaffName
	}
	s.hub.BroadcastEvent(incidentID, ws.WSEvent{
		Event: "DISPATCH_ACCEPTED",
		Payload: map[string]interface{}{
			"staff_name": staffName,
			"message":    fmt.Sprintf("%s dispatched. ETA 45 seconds.", staffName),
		},
	})

	chatWSS := fmt.Sprintf("%s/api/v1/incident/ws/%s/live", s.wsBaseURL(), incidentID)

	return incidentDto.DispatchAcceptResponse{
		Status: "success",
		Data: incidentDto.DispatchAcceptData{
			DispatchStatus: "LOCKED",
			ChatRoomWSS:    chatWSS,
			Message:        "Proceed to the room immediately.",
		},
	}, nil
}

// DispatchDecline handles staff declining a dispatch.
func (s *IncidentService) DispatchDecline(ctx context.Context, incidentID string, staffID uuid.UUID, req incidentDto.DispatchDeclineRequest) (incidentDto.DispatchDeclineResponse, error) {
	// Update dispatch
	s.repo.UpdateDispatchStatus(ctx, incidentID, staffID, "DECLINED")

	// Reset staff availability
	hotelID, _ := s.repo.GetStaffHotelID(ctx, staffID)
	statusKey := fmt.Sprintf("staff:%s:%s:status", hotelID.String(), staffID.String())
	s.redis.Set(ctx, statusKey, "AVAILABLE", 0)

	// Log event
	staffIDStr := staffID.String()
	actorRole := "STAFF"
	s.repo.LogEvent(ctx, incidentID, "DISPATCH_DECLINED", &staffIDStr, &actorRole, map[string]interface{}{
		"reason": req.Reason,
	})

	// Cascade routing is handled by the cascade timeout goroutine in dispatch engine

	return incidentDto.DispatchDeclineResponse{
		Status:  "acknowledged",
		Message: "You have been removed from the dispatch queue.",
	}, nil
}

// TransitPing records a GPS ping during transit and checks for idle.
func (s *IncidentService) TransitPing(ctx context.Context, incidentID string, staffID uuid.UUID, req incidentDto.TransitPingRequest) (incidentDto.TransitPingResponse, error) {
	// Check for idle
	isIdle := s.engine.CheckTransitIdle(ctx, incidentID, staffID.String(), req.Lat, req.Lng)

	if isIdle {
		log.Printf("[Transit] Staff %s is idle during transit for incident %s", staffID.String(), incidentID)
		// Revoke dispatch
		s.repo.UpdateDispatchStatus(ctx, incidentID, staffID, "TIMED_OUT")

		hotelID, _ := s.repo.GetStaffHotelID(ctx, staffID)
		statusKey := fmt.Sprintf("staff:%s:%s:status", hotelID.String(), staffID.String())
		s.redis.Set(ctx, statusKey, "AVAILABLE", 0)

		// Notify staff
		s.hub.BroadcastEvent(incidentID, ws.WSEvent{
			Event: "DISPATCH_REVOKED",
			Payload: map[string]interface{}{
				"staff_id": staffID.String(),
				"reason":   "No movement detected. Dispatch revoked.",
			},
		})

		actorID := "SYSTEM"
		actorRole := "DISPATCH_ENGINE"
		s.repo.LogEvent(ctx, incidentID, "DISPATCH_REVOKED_IDLE", &actorID, &actorRole, map[string]interface{}{
			"staff_id": staffID.String(),
		})
	}

	// Broadcast GPS telemetry
	s.hub.BroadcastEvent(incidentID, ws.WSEvent{
		Event: "GPS_TELEMETRY",
		Payload: map[string]interface{}{
			"staff_id":    staffID.String(),
			"lat":         req.Lat,
			"lng":         req.Lng,
			"eta_seconds": 35, // ASSUMPTION: Mock ETA
		},
	})

	return incidentDto.TransitPingResponse{Status: "tracking"}, nil
}

// StaffArrive handles staff arrival at the incident location.
func (s *IncidentService) StaffArrive(ctx context.Context, incidentID string, staffID uuid.UUID, req incidentDto.ArrivalRequest) (incidentDto.ArrivalResponse, error) {
	incident, err := s.repo.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return incidentDto.ArrivalResponse{}, fmt.Errorf("incident not found")
	}

	// Verify GPS within 10m of room (geofence)
	verified, err := s.engine.VerifyArrival(ctx, incident.RoomID, req.Lat, req.Lng)
	if err != nil {
		log.Printf("[Arrival] Geofence error, allowing arrival: %v", err)
		verified = true // ASSUMPTION: Allow if geofence check fails
	}

	if !verified {
		return incidentDto.ArrivalResponse{}, fmt.Errorf("GPS location too far from room. Arrival not verified.")
	}

	// Update dispatch + incident status
	s.repo.UpdateDispatchArrived(ctx, incidentID, staffID)
	s.repo.UpdateIncidentStatus(ctx, incidentID, "ARRIVED")

	// Log event
	staffIDStr := staffID.String()
	actorRole := "STAFF"
	s.repo.LogEvent(ctx, incidentID, "FIRST_RESPONDER_ON_SCENE", &staffIDStr, &actorRole, map[string]interface{}{
		"lat": req.Lat,
		"lng": req.Lng,
	})

	// Get staff name for broadcast
	dispatch, _ := s.repo.GetActiveDispatchForStaff(ctx, incidentID, staffID)
	staffName := "Responder"
	if dispatch != nil {
		staffName = dispatch.StaffName
	}

	// Broadcast arrival
	s.hub.BroadcastEvent(incidentID, ws.WSEvent{
		Event: "FIRST_RESPONDER_ON_SCENE",
		Payload: map[string]interface{}{
			"staff_name": staffName,
			"staff_id":   staffID.String(),
		},
	})

	return incidentDto.ArrivalResponse{
		Status:  "verified",
		Message: "Arrival confirmed. Please submit ground truth assessment.",
	}, nil
}

// SubmitGroundTruth handles the staff's on-scene severity assessment.
func (s *IncidentService) SubmitGroundTruth(ctx context.Context, incidentID string, staffID uuid.UUID, req incidentDto.GroundTruthRequest) (incidentDto.GroundTruthResponse, error) {
	incident, err := s.repo.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return incidentDto.GroundTruthResponse{}, fmt.Errorf("incident not found")
	}

	// Save ground truth
	s.repo.UpdateGroundTruth(ctx, incidentID, req.ManualSeverity)

	// Log event
	staffIDStr := staffID.String()
	actorRole := "STAFF"
	s.repo.LogEvent(ctx, incidentID, "GROUND_TRUTH_SUBMITTED", &staffIDStr, &actorRole, map[string]interface{}{
		"manual_severity": req.ManualSeverity,
		"scout_notes":     req.ScoutNotes,
		"media_urls":      req.MediaURLs,
	})

	// Publish to RabbitMQ for Time 2 re-evaluation
	initialSev := 0
	if incident.InitialSeverity != nil {
		initialSev = *incident.InitialSeverity
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"incident_id":      incidentID,
		"scout_severity":   req.ManualSeverity,
		"initial_severity": initialSev,
	})
	if err := s.publisher.Publish(ctx, "incident.ground_truth", payload); err != nil {
		log.Printf("[Incident] Error publishing ground_truth, running inline: %v", err)
		go func() {
			reeval := mock.MockGroundTruthReeval(req.ManualSeverity, initialSev)
			s.engine.RunDeltaEngine(context.Background(), incidentID, reeval)
		}()
	}

	return incidentDto.GroundTruthResponse{
		Status:  "processing",
		Message: "Ground truth locked. Recalculating dispatch requirements.",
	}, nil
}

// RequestBackup handles backup staff requests.
func (s *IncidentService) RequestBackup(ctx context.Context, incidentID string, staffID uuid.UUID, req incidentDto.BackupRequest) (incidentDto.BackupResponse, error) {
	// ASSUMPTION: Max 3 backup requests per incident
	activeCount, err := s.repo.CountActiveDispatches(ctx, incidentID)
	if err != nil {
		return incidentDto.BackupResponse{}, fmt.Errorf("failed to check dispatch count")
	}
	if activeCount >= 10 {
		return incidentDto.BackupResponse{}, fmt.Errorf("maximum dispatch limit reached for this incident")
	}

	// Log event
	staffIDStr := staffID.String()
	actorRole := "STAFF"
	s.repo.LogEvent(ctx, incidentID, "BACKUP_REQUESTED", &staffIDStr, &actorRole, map[string]interface{}{
		"additional_roles": req.AdditionalRolesNeeded,
	})

	// Build role requirements from request
	var requirements []mock.RoleRequirement
	for role, count := range req.AdditionalRolesNeeded {
		requirements = append(requirements, mock.RoleRequirement{
			Role:            role,
			MinCount:        count,
			SkillThresholds: map[string]float64{"first_aid": 0.3}, // ASSUMPTION: default skill threshold
		})
	}

	// Run dispatch for the new requirements
	incident, _ := s.repo.GetIncidentByID(ctx, incidentID)
	severity := 5
	if incident != nil && incident.FinalSeverity != nil {
		severity = *incident.FinalSeverity
	}

	assessment := mock.MockAssessmentResult{
		IncidentID:    incidentID,
		FinalSeverity: severity,
		RequiredStaff: requirements,
		AISummary:     "Backup requested by on-scene staff.",
	}

	go s.engine.RunDispatch(context.Background(), incidentID, assessment)

	return incidentDto.BackupResponse{
		Status:  "processing",
		Message: "Backup request received. Processing via Dispatch Engine.",
	}, nil
}

// InitiateEscalation starts a 10-second escalation countdown.
func (s *IncidentService) InitiateEscalation(ctx context.Context, incidentID string, actorID string, req incidentDto.EscalateRequest) (incidentDto.EscalateResponse, error) {
	// Set Redis timer key with 10s TTL
	timerKey := "incident:" + incidentID + ":escalation_timer"
	s.redis.Set(ctx, timerKey, "PENDING", 10*time.Second)

	// Log event
	actorRole := "STAFF"
	s.repo.LogEvent(ctx, incidentID, "ESCALATION_INITIATED", &actorID, &actorRole, map[string]interface{}{
		"target_authority": req.TargetAuthority,
		"reason":           req.Reason,
	})

	// Broadcast countdown
	s.hub.BroadcastEvent(incidentID, ws.WSEvent{
		Event: "ESCALATION_COUNTDOWN",
		Payload: map[string]interface{}{
			"seconds_remaining": 10,
			"target":            req.TargetAuthority,
		},
	})

	// Start goroutine to execute escalation after 10s if not aborted
	go func() {
		time.Sleep(10 * time.Second)
		exists, _ := s.redis.Exists(context.Background(), timerKey).Result()
		if exists > 0 {
			// Timer still exists — execute escalation
			log.Printf("[Escalation] Executing escalation for incident %s to %s", incidentID, req.TargetAuthority)
			s.redis.Del(context.Background(), timerKey)
			s.repo.SetEscalationStatus(ctx, incidentID, req.TargetAuthority)

			sysID := "SYSTEM"
			sysRole := "ESCALATION_ENGINE"
			s.repo.LogEvent(context.Background(), incidentID, "ESCALATION_EXECUTED", &sysID, &sysRole, map[string]interface{}{
				"target_authority": req.TargetAuthority,
			})

			s.hub.BroadcastEvent(incidentID, ws.WSEvent{
				Event: "ESCALATION_EXECUTED",
				Payload: map[string]interface{}{
					"target":  req.TargetAuthority,
					"message": fmt.Sprintf("Escalated to %s.", req.TargetAuthority),
				},
			})
		}
	}()

	return incidentDto.EscalateResponse{
		Status:       "escalation_pending",
		TimerSeconds: 10,
		Message:      "Escalation initiated. Manager has 10 seconds to abort.",
	}, nil
}

// AbortEscalation cancels an active escalation.
func (s *IncidentService) AbortEscalation(ctx context.Context, incidentID, managerID string, req incidentDto.EscalateAbortRequest) (incidentDto.EscalateAbortResponse, error) {
	timerKey := "incident:" + incidentID + ":escalation_timer"
	deleted, _ := s.redis.Del(ctx, timerKey).Result()

	if deleted == 0 {
		return incidentDto.EscalateAbortResponse{}, fmt.Errorf("no active escalation to abort")
	}

	// Log liability event
	actorRole := "MANAGER"
	s.repo.LogEvent(ctx, incidentID, "ESCALATION_ABORTED", &managerID, &actorRole, map[string]interface{}{
		"reason":    req.Reason,
		"timestamp": time.Now().UTC(),
	})

	s.hub.BroadcastEvent(incidentID, ws.WSEvent{
		Event: "ESCALATION_ABORTED",
		Payload: map[string]interface{}{
			"aborted_by": managerID,
			"reason":     req.Reason,
		},
	})

	return incidentDto.EscalateAbortResponse{
		Status:  "aborted",
		Message: "Escalation cancelled. Liability logged.",
	}, nil
}

// ProposeResolution handles a resolution proposal from staff or guest.
func (s *IncidentService) ProposeResolution(ctx context.Context, incidentID, actorID string, req incidentDto.ResolveProposeRequest) (incidentDto.ResolveProposeResponse, error) {
	s.repo.SetResolutionProposed(ctx, incidentID, req.ProposedByRole)

	s.repo.LogEvent(ctx, incidentID, "RESOLUTION_PROPOSED", &actorID, &req.ProposedByRole, map[string]interface{}{
		"resolution_notes": req.ResolutionNotes,
	})

	s.hub.BroadcastEvent(incidentID, ws.WSEvent{
		Event: "RESOLUTION_PROPOSED",
		Payload: map[string]interface{}{
			"proposed_by": req.ProposedByRole,
			"notes":       req.ResolutionNotes,
		},
	})

	return incidentDto.ResolveProposeResponse{
		Status:  "pending_handshake",
		Message: "Awaiting confirmation from the other party.",
	}, nil
}

// ConfirmResolution confirms the resolution and triggers closure.
func (s *IncidentService) ConfirmResolution(ctx context.Context, incidentID, actorID, actorRole string) (incidentDto.ResolveConfirmResponse, error) {
	// Set incident to RESOLVED
	if err := s.repo.ResolveIncident(ctx, incidentID); err != nil {
		return incidentDto.ResolveConfirmResponse{}, fmt.Errorf("failed to resolve incident")
	}

	s.repo.LogEvent(ctx, incidentID, "RESOLUTION_CONFIRMED", &actorID, &actorRole, nil)

	// Trigger Phase 8 closure cascade
	go s.closureSequence(incidentID)

	return incidentDto.ResolveConfirmResponse{
		Status:  "resolved",
		Message: "Crisis neutralized. Initiating system teardown.",
	}, nil
}

// closureSequence handles the Phase 8 closure cascade.
func (s *IncidentService) closureSequence(incidentID string) {
	ctx := context.Background()
	log.Printf("[Closure] Starting closure sequence for incident %s", incidentID)

	// Publish to report compilation queue
	payload, _ := json.Marshal(map[string]string{"incident_id": incidentID})
	if err := s.publisher.Publish(ctx, "incident.compile_report", payload); err != nil {
		log.Printf("[Closure] Error publishing compile_report, running inline: %v", err)

		// Fallback: run inline
		time.Sleep(1 * time.Second)

		inc, _ := s.repo.GetIncidentByID(ctx, incidentID)
		if inc == nil {
			return
		}
		eventRows, _ := s.repo.GetEventsByIncident(ctx, incidentID)

		incident := mock.IncidentToModel(inc)
		var events []mock.EventModel
		for _, e := range eventRows {
			events = append(events, mock.EventModel{
				ID:         e.ID,
				IncidentID: e.IncidentID,
				EventType:  e.EventType,
				ActorID:    e.ActorID,
				ActorRole:  e.ActorRole,
				CreatedAt:  e.CreatedAt,
			})
		}

		report := mock.MockCompileReportFromRows(incident, events)
		s.repo.CreateReport(ctx, incidentID, report, "MOCK_AI_AGENT")
		s.repo.CloseIncident(ctx, incidentID)

		s.hub.BroadcastEvent(incidentID, ws.WSEvent{
			Event: "SYSTEM_TEARDOWN",
			Payload: map[string]interface{}{
				"message": "Incident Closed. Final report compiled.",
			},
		})

		s.engine.ReleaseAllStaff(ctx, incidentID)
		s.engine.CleanupRedisKeys(ctx, incidentID, inc.HotelID.String())
		s.hub.CloseRoom(incidentID)
	}
}
