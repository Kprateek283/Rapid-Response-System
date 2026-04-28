package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 8192
)

// Client represents a single WebSocket connection in a room.
type Client struct {
	id   string
	role string // GUEST, STAFF, MANAGER
	conn *websocket.Conn
	send chan []byte

	// OnMessage is called when the client sends a message upstream.
	// Set by the room/handler to process incoming events.
	OnMessage func(clientID, role string, data []byte)
}

// NewClient creates a new WebSocket client.
func NewClient(id, role string, conn *websocket.Conn) *Client {
	return &Client{
		id:   id,
		role: role,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

// readPump pumps messages from the WebSocket connection to the room.
func (c *Client) readPump(room *Room) {
	defer func() {
		room.mu.Lock()
		delete(room.clients, c.id)
		room.mu.Unlock()
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[WS Client] Read error for %s: %v", c.id, err)
			}
			break
		}

		// Parse the incoming event
		var event WSEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("[WS Client] Invalid JSON from %s: %v", c.id, err)
			continue
		}

		// Call the handler if set
		if c.OnMessage != nil {
			c.OnMessage(c.id, c.role, message)
		}

		// For chat messages, broadcast to room (will be processed by handler for translation)
		if event.Event == "CHAT_MESSAGE" {
			room.broadcast <- message
		}
	}
}

// writePump pumps messages from the send channel to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed — send close frame
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain queued messages into the current write
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
