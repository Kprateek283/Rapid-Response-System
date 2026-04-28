package incident

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google-hackathon/rapid-response/internal/api/middleware"
	"github.com/google-hackathon/rapid-response/internal/models"
	"github.com/google-hackathon/rapid-response/internal/ws"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // ASSUMPTION: Accept all origins for hackathon
	},
}

// WSHandler handles WebSocket upgrade requests.
type WSHandler struct {
	hub *ws.Hub
}

// NewWSHandler creates a new WebSocket handler.
func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// HandleWSUpgrade handles GET /incident/ws/:incident_id/live
func (h *WSHandler) HandleWSUpgrade(w http.ResponseWriter, r *http.Request) {
	incidentID := chi.URLParam(r, "incident_id")
	if incidentID == "" {
		http.Error(w, "incident_id required", http.StatusBadRequest)
		return
	}

	// Extract claims from context (set by auth middleware)
	claims, ok := r.Context().Value(middleware.ClaimsKey).(*models.UserClaims)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	// Determine client ID based on role
	clientID := claims.UserID
	if claims.GuestID != "" {
		clientID = claims.GuestID
	}

	client := ws.NewClient(clientID, claims.Role, conn)

	// Set up message handler for chat translations etc.
	client.OnMessage = func(cID, role string, data []byte) {
		log.Printf("[WS] Message from %s (%s): %s", cID, role, string(data))
		// Chat message handling is done in client.readPump via broadcast
	}

	h.hub.JoinRoom(incidentID, client)

	log.Printf("[WS] Client %s (%s) connected to incident %s", clientID, claims.Role, incidentID)
}
