package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google-hackathon/rapid-response/internal/dispatch"
	"github.com/google-hackathon/rapid-response/internal/mock"
	"github.com/google-hackathon/rapid-response/internal/models"
	incidentRepo "github.com/google-hackathon/rapid-response/internal/repository/incident"
	"github.com/google-hackathon/rapid-response/internal/ws"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer processes messages from RabbitMQ queues using mock AI functions.
type Consumer struct {
	conn     *amqp.Connection
	repo     *incidentRepo.IncidentRepository
	engine   *dispatch.Engine
	hub      *ws.Hub
}

// NewConsumer creates a new RabbitMQ consumer.
func NewConsumer(conn *amqp.Connection, repo *incidentRepo.IncidentRepository, engine *dispatch.Engine, hub *ws.Hub) *Consumer {
	return &Consumer{
		conn:   conn,
		repo:   repo,
		engine: engine,
		hub:    hub,
	}
}

// StartConsumers declares queues and starts goroutine consumers for each.
func (c *Consumer) StartConsumers() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	// Consumer 1: AI Assessment (after media-ready)
	go c.consumeQueue(ch, "ai_assessment_queue", c.handleAIAssessment)

	// Consumer 2: Ground Truth Re-evaluation
	go c.consumeQueue(ch, "ground_truth_queue", c.handleGroundTruth)

	// Consumer 3: Report Compilation
	go c.consumeQueue(ch, "report_compilation_queue", c.handleReportCompilation)

	log.Println("[MQ Consumer] All consumers started")
	return nil
}

func (c *Consumer) consumeQueue(ch *amqp.Channel, queueName string, handler func([]byte)) {
	msgs, err := ch.Consume(queueName, "", true, false, false, false, nil)
	if err != nil {
		log.Printf("[MQ Consumer] Error consuming %s: %v", queueName, err)
		return
	}

	log.Printf("[MQ Consumer] Listening on queue: %s", queueName)
	for msg := range msgs {
		handler(msg.Body)
	}
}

// handleAIAssessment processes the Time 1 AI assessment after media upload.
func (c *Consumer) handleAIAssessment(body []byte) {
	var payload struct {
		IncidentID string `json:"incident_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[MQ Consumer] Error parsing assessment payload: %v", err)
		return
	}

	log.Printf("[MQ Consumer] Processing AI assessment for incident %s", payload.IncidentID)

	// Get incident details
	incRow, err := c.repo.GetIncidentByID(context.Background(), payload.IncidentID)
	if err != nil {
		log.Printf("[MQ Consumer] Error getting incident: %v", err)
		return
	}

	// Build a models.Incident for the mock
	incident := models.Incident{
		ID:      incRow.ID,
		HotelID: incRow.HotelID,
		RoomID:  incRow.RoomID,
		GuestID: incRow.GuestID,
	}

	// Call mock AI assessment (sleeps 2s to simulate processing)
	assessment := mock.MockAIAssessment(incident)

	// Broadcast AI result via WS
	c.hub.BroadcastEvent(payload.IncidentID, ws.WSEvent{
		Event: "AI_ASSESSMENT_COMPLETE",
		Payload: map[string]interface{}{
			"severity":   assessment.FinalSeverity,
			"ai_summary": assessment.AISummary,
		},
	})

	// Feed to dispatch engine
	c.engine.RunDispatch(context.Background(), payload.IncidentID, assessment)
}

// handleGroundTruth processes the Time 2 re-evaluation after ground truth.
func (c *Consumer) handleGroundTruth(body []byte) {
	var payload struct {
		IncidentID      string `json:"incident_id"`
		ScoutSeverity   int    `json:"scout_severity"`
		InitialSeverity int    `json:"initial_severity"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[MQ Consumer] Error parsing ground truth payload: %v", err)
		return
	}

	log.Printf("[MQ Consumer] Processing ground truth re-eval for incident %s", payload.IncidentID)

	// Call mock re-evaluation
	reeval := mock.MockGroundTruthReeval(payload.ScoutSeverity, payload.InitialSeverity)

	// Run delta engine
	c.engine.RunDeltaEngine(context.Background(), payload.IncidentID, reeval)

	// Broadcast update
	c.hub.BroadcastEvent(payload.IncidentID, ws.WSEvent{
		Event: "AI_REEVAL_COMPLETE",
		Payload: map[string]interface{}{
			"final_severity": reeval.FinalSeverity,
			"ai_summary":     reeval.AISummary,
		},
	})
}

// handleReportCompilation processes the final report compilation.
func (c *Consumer) handleReportCompilation(body []byte) {
	var payload struct {
		IncidentID string `json:"incident_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[MQ Consumer] Error parsing report payload: %v", err)
		return
	}

	log.Printf("[MQ Consumer] Compiling report for incident %s", payload.IncidentID)

	ctx := context.Background()

	// Get incident
	incRow, err := c.repo.GetIncidentByID(ctx, payload.IncidentID)
	if err != nil {
		log.Printf("[MQ Consumer] Error: %v", err)
		return
	}

	// Get events
	eventRows, err := c.repo.GetEventsByIncident(ctx, payload.IncidentID)
	if err != nil {
		log.Printf("[MQ Consumer] Error getting events: %v", err)
		return
	}

	// Convert to model types for the mock
	incident := models.Incident{
		ID:            incRow.ID,
		HotelID:       incRow.HotelID,
		RoomID:        incRow.RoomID,
		GuestID:       incRow.GuestID,
		TriggerSource: incRow.TriggerSource,
		FinalSeverity: incRow.FinalSeverity,
		AISummary:     incRow.AISummary,
	}

	events := make([]models.IncidentEvent, len(eventRows))
	for i, e := range eventRows {
		events[i] = models.IncidentEvent{
			ID:         e.ID,
			IncidentID: e.IncidentID,
			EventType:  e.EventType,
			ActorID:    e.ActorID,
			ActorRole:  e.ActorRole,
			CreatedAt:  e.CreatedAt,
		}
	}

	// Mock compile report
	report := mock.MockCompileReport(incident, events)

	// Save report
	if err := c.repo.CreateReport(ctx, payload.IncidentID, report, "MOCK_AI_AGENT"); err != nil {
		log.Printf("[MQ Consumer] Error saving report: %v", err)
		return
	}

	// Close incident
	if err := c.repo.CloseIncident(ctx, payload.IncidentID); err != nil {
		log.Printf("[MQ Consumer] Error closing incident: %v", err)
	}

	// Broadcast teardown
	c.hub.BroadcastEvent(payload.IncidentID, ws.WSEvent{
		Event: "SYSTEM_TEARDOWN",
		Payload: map[string]interface{}{
			"message":    "Incident Closed. Final report compiled.",
			"report_url": fmt.Sprintf("/api/v1/incident/%s/report", payload.IncidentID),
		},
	})

	// Release all staff
	c.engine.ReleaseAllStaff(ctx, payload.IncidentID)

	// Cleanup Redis
	c.engine.CleanupRedisKeys(ctx, payload.IncidentID, incRow.HotelID.String())

	// Close WS room after a short delay to let clients receive teardown
	go func() {
		time.Sleep(2 * time.Second)
		c.hub.CloseRoom(payload.IncidentID)
	}()

	log.Printf("[MQ Consumer] Incident %s fully closed", payload.IncidentID)
}

