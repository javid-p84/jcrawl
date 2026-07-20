package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow same-origin and localhost connections
		// TODO: Configure for production
		return true
	},
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	clients       map[string]map[*Client]bool // userID -> clients
	broadcast     chan *NotificationMessage
	register      chan *Client
	unregister    chan *Client
	validateToken func(token string) (string, error)
	mu            sync.RWMutex
}

// Client represents a WebSocket connection
type Client struct {
	UserID string
	Hub    *WebSocketHub
	Conn   *websocket.Conn
	Send   chan *NotificationMessage
}

// NotificationMessage is sent over WebSocket
type NotificationMessage struct {
	Type      string      `json:"type"`      // "notification", "status", etc
	UserID    string      `json:"user_id"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// NewWebSocketHub creates a new WebSocket hub. validateToken authenticates
// connecting clients (token -> user ID); typically AuthService.ValidateToken.
func NewWebSocketHub(validateToken func(token string) (string, error)) *WebSocketHub {
	return &WebSocketHub{
		clients:       make(map[string]map[*Client]bool),
		broadcast:     make(chan *NotificationMessage, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		validateToken: validateToken,
	}
}

// Start runs the WebSocket hub
func (h *WebSocketHub) Start() {
	go func() {
		for {
			select {
			case client := <-h.register:
				h.mu.Lock()
				if _, ok := h.clients[client.UserID]; !ok {
					h.clients[client.UserID] = make(map[*Client]bool)
				}
				h.clients[client.UserID][client] = true
				h.mu.Unlock()
				log.Printf("WebSocket client registered: %s\n", client.UserID)

			case client := <-h.unregister:
				h.mu.Lock()
				if clients, ok := h.clients[client.UserID]; ok {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						close(client.Send)
						if len(clients) == 0 {
							delete(h.clients, client.UserID)
						}
					}
				}
				h.mu.Unlock()
				log.Printf("WebSocket client unregistered: %s\n", client.UserID)

			case message := <-h.broadcast:
				h.mu.RLock()
				if clients, ok := h.clients[message.UserID]; ok {
					for client := range clients {
						select {
						case client.Send <- message:
						default:
							// Client's send channel is full, close it
							close(client.Send)
							delete(clients, client)
						}
					}
				}
				h.mu.RUnlock()
			}
		}
	}()
}

// BroadcastNotification sends a notification to user via WebSocket
func (h *WebSocketHub) BroadcastNotification(userID string, notif *models.Notification) {
	message := &NotificationMessage{
		Type:      "notification",
		UserID:    userID,
		Data:      notif,
		Timestamp: notif.CreatedAt.Unix(),
	}
	h.broadcast <- message
}

// HandleWebSocket handles a WebSocket connection.
// Browsers cannot set custom headers on WebSocket connects, so the JWT is
// passed as a query parameter: /ws/notifications?token=<jwt>
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if token == "" {
		http.Error(w, "token query parameter or Authorization header required", http.StatusUnauthorized)
		return
	}

	userID, err := h.validateToken(token)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	client := &Client{
		UserID: userID,
		Hub:    h,
		Conn:   conn,
		Send:   make(chan *NotificationMessage, 256),
	}
	h.register <- client

	go client.readPump()
	go client.writePump()
}

// readPump reads messages from WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v\n", err)
			}
			return
		}

		// Process incoming message (e.g., acknowledge notification)
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			log.Printf("WebSocket message from %s: %v\n", c.UserID, msg)
		}
	}
}

// writePump writes messages to WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
