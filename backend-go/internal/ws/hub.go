package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub manages all WebSocket rooms, one per active incident.
type Hub struct {
	rooms map[string]*Room // incident_id -> Room
	mu    sync.RWMutex
}

// NewHub creates a new WebSocket Hub.
func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
	}
}

// Room represents a WebSocket room for a single incident.
type Room struct {
	incidentID string
	clients    map[string]*Client // client_id -> Client
	broadcast  chan []byte
	mu         sync.RWMutex
	done       chan struct{}
}

// WSEvent is the standard event envelope for WebSocket messages.
type WSEvent struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// CreateRoom creates a new room for an incident and starts the broadcast loop.
func (h *Hub) CreateRoom(incidentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[incidentID]; exists {
		return // Room already exists
	}

	room := &Room{
		incidentID: incidentID,
		clients:    make(map[string]*Client),
		broadcast:  make(chan []byte, 256),
		done:       make(chan struct{}),
	}

	h.rooms[incidentID] = room

	// Start the broadcast goroutine
	go room.runBroadcast()

	log.Printf("[WS Hub] Room created for incident %s", incidentID)
}

// JoinRoom adds a client to an incident room.
func (h *Hub) JoinRoom(incidentID string, client *Client) {
	h.mu.RLock()
	room, exists := h.rooms[incidentID]
	h.mu.RUnlock()

	if !exists {
		log.Printf("[WS Hub] Room %s not found, creating it", incidentID)
		h.CreateRoom(incidentID)
		h.mu.RLock()
		room = h.rooms[incidentID]
		h.mu.RUnlock()
	}

	room.mu.Lock()
	room.clients[client.id] = client
	room.mu.Unlock()

	log.Printf("[WS Hub] Client %s (%s) joined room %s", client.id, client.role, incidentID)

	// Start client read/write pumps
	go client.writePump()
	go client.readPump(room)
}

// LeaveRoom removes a client from an incident room.
func (h *Hub) LeaveRoom(incidentID, clientID string) {
	h.mu.RLock()
	room, exists := h.rooms[incidentID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	room.mu.Lock()
	if client, ok := room.clients[clientID]; ok {
		close(client.send)
		delete(room.clients, clientID)
	}
	room.mu.Unlock()

	log.Printf("[WS Hub] Client %s left room %s", clientID, incidentID)
}

// BroadcastToRoom sends a message to all clients in an incident room.
func (h *Hub) BroadcastToRoom(incidentID string, message []byte) {
	h.mu.RLock()
	room, exists := h.rooms[incidentID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	select {
	case room.broadcast <- message:
	default:
		log.Printf("[WS Hub] Broadcast channel full for room %s, dropping message", incidentID)
	}
}

// BroadcastEvent is a convenience method to broadcast a typed event.
func (h *Hub) BroadcastEvent(incidentID string, event WSEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[WS Hub] Failed to marshal event: %v", err)
		return
	}
	h.BroadcastToRoom(incidentID, data)
}

// CloseRoom closes all client connections and removes the room.
func (h *Hub) CloseRoom(incidentID string) {
	h.mu.Lock()
	room, exists := h.rooms[incidentID]
	if !exists {
		h.mu.Unlock()
		return
	}
	delete(h.rooms, incidentID)
	h.mu.Unlock()

	// Signal broadcast goroutine to stop
	close(room.done)

	// Close all client connections
	room.mu.Lock()
	for id, client := range room.clients {
		close(client.send)
		client.conn.Close()
		delete(room.clients, id)
	}
	room.mu.Unlock()

	log.Printf("[WS Hub] Room closed for incident %s", incidentID)
}

// RoomExists checks if a room exists for the given incident.
func (h *Hub) RoomExists(incidentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.rooms[incidentID]
	return exists
}

// GetRoomClientCount returns the number of clients in a room.
func (h *Hub) GetRoomClientCount(incidentID string) int {
	h.mu.RLock()
	room, exists := h.rooms[incidentID]
	h.mu.RUnlock()
	if !exists {
		return 0
	}
	room.mu.RLock()
	defer room.mu.RUnlock()
	return len(room.clients)
}

// runBroadcast is the broadcast loop for a room — runs as a goroutine.
func (r *Room) runBroadcast() {
	for {
		select {
		case message, ok := <-r.broadcast:
			if !ok {
				return
			}
			r.mu.RLock()
			for _, client := range r.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, skip
				}
			}
			r.mu.RUnlock()
		case <-r.done:
			return
		}
	}
}
