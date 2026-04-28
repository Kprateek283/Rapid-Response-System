package incident

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	incidentRepo "github.com/google-hackathon/rapid-response/internal/repository/incident"
	"github.com/google-hackathon/rapid-response/internal/ws"
	"github.com/google/uuid"
)

// SMSHandler handles Twilio SMS webhook for incident triggers.
type SMSHandler struct {
	repo *incidentRepo.IncidentRepository
	hub  *ws.Hub
}

// NewSMSHandler creates a new SMS webhook handler.
func NewSMSHandler(repo *incidentRepo.IncidentRepository, hub *ws.Hub) *SMSHandler {
	return &SMSHandler{repo: repo, hub: hub}
}

// HandleSMSWebhook handles POST /incident/trigger/sms-webhook
// ASSUMPTION: Twilio signature validation is mocked (accept all for hackathon)
func (h *SMSHandler) HandleSMSWebhook(w http.ResponseWriter, r *http.Request) {
	// Parse URL-encoded form data from Twilio
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, twiMLError("Invalid request"))
		return
	}

	body := r.FormValue("Body")
	from := r.FormValue("From")

	log.Printf("[SMS] Received SMS from %s: %s", from, body)

	// Parse: SOS|ROOM:101|LAT:12.96|LNG:77.74|BLE_PRESS
	parts := strings.Split(body, "|")
	if len(parts) < 1 || parts[0] != "SOS" {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, twiMLResponse("Invalid SOS format. Use: SOS|ROOM:101|LAT:12.96|LNG:77.74"))
		return
	}

	var roomNumber string
	var lat, lng float64

	for _, part := range parts[1:] {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "ROOM":
			roomNumber = kv[1]
		case "LAT":
			fmt.Sscanf(kv[1], "%f", &lat)
		case "LNG":
			fmt.Sscanf(kv[1], "%f", &lng)
		}
	}

	if roomNumber == "" {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, twiMLResponse("Room number required. Use: SOS|ROOM:101|LAT:12.96|LNG:77.74"))
		return
	}

	// ASSUMPTION: For SMS triggers, we use a placeholder hotel/room/guest
	// In production, we'd look up the room and find the associated hotel
	hotelID := uuid.New() // Placeholder
	roomID := uuid.New()  // Placeholder
	guestID := "sms_" + from

	incidentID, err := h.repo.CreateIncident(r.Context(), hotelID, roomID, guestID, "SMS_WEBHOOK", &lat, &lng)
	if err != nil {
		log.Printf("[SMS] Error creating incident: %v", err)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, twiMLResponse("Error processing SOS. Please call emergency services directly."))
		return
	}

	// Create WS room
	h.hub.CreateRoom(incidentID)

	// Log event
	actorRole := "GUEST"
	h.repo.LogEvent(r.Context(), incidentID, "INCIDENT_CREATED_VIA_SMS", &guestID, &actorRole, map[string]interface{}{
		"from":        from,
		"room_number": roomNumber,
		"raw_body":    body,
	})

	log.Printf("[SMS] Created incident %s from SMS", incidentID)

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, twiMLResponse(fmt.Sprintf("SOS received. Incident %s created. Help is on the way.", incidentID)))
}

func twiMLResponse(message string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Message>%s</Message>
</Response>`, message)
}

func twiMLError(message string) string {
	return twiMLResponse("Error: " + message)
}
