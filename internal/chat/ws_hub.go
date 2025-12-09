package chat

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"loveguru/internal/db"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Message struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id"`
	SenderID  string      `json:"sender_id"`
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

type Client struct {
	ID        string
	Conn      *websocket.Conn
	Send      chan Message
	SessionID string
	UserID    string
}

type Hub struct {
	clients    map[string]*Client
	clientLock sync.RWMutex
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	service    *Service
	ctx        context.Context
}

func NewHub(service *Service) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		service:    service,
		ctx:        context.Background(),
	}
}

func (h *Hub) Run() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-ticker.C:
			h.cleanupConnections()
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	h.clients[client.ID] = client

	// Send recent messages to the newly connected client
	go h.sendRecentMessages(client)
}

func (h *Hub) unregisterClient(client *Client) {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
	}
}

func (h *Hub) broadcastMessage(message Message) {
	h.clientLock.RLock()
	defer h.clientLock.RUnlock()

	for _, client := range h.clients {
		if client.SessionID == message.SessionID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client.ID)
			}
		}
	}
}

func (h *Hub) sendRecentMessages(client *Client) {
	// Get recent messages from database
	messages, err := h.service.repo.GetMessages(h.ctx, db.GetMessagesParams{
		SessionID: uuid.MustParse(client.SessionID),
		Limit:     50,
		Offset:    0,
	})
	if err != nil {
		log.Printf("Error getting recent messages: %v", err)
		return
	}

	for _, msg := range messages {
		message := Message{
			Type:      "MESSAGE",
			SessionID: client.SessionID,
			SenderID:  msg.SenderID.String(),
			Content:   msg.Content,
			Timestamp: msg.CreatedAt.Time,
		}

		select {
		case client.Send <- message:
		case <-time.After(5 * time.Second):
			return
		}
	}
}

func (h *Hub) cleanupConnections() {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	for id, client := range h.clients {
		if err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
			close(client.Send)
			delete(h.clients, id)
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, sessionID, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	clientID := uuid.New().String()
	client := &Client{
		ID:        clientID,
		Conn:      conn,
		Send:      make(chan Message, 256),
		SessionID: sessionID,
		UserID:    userID,
	}

	h.register <- client
	defer func() {
		h.unregister <- client
		conn.Close()
	}()

	// Start goroutines for reading and writing
	go h.writePump(client)
	h.readPump(client)
}

func (h *Hub) writePump(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) readPump(client *Client) {
	defer client.Conn.Close()

	for {
		var msg Message
		if err := client.Conn.ReadJSON(&msg); err != nil {
			break
		}

		// Validate and process the message
		if msg.Type == "MESSAGE" && msg.Content != "" {
			// Store message in database
			err := h.service.InsertMessage(h.ctx, client.SessionID, "USER", client.UserID, msg.Content)
			if err != nil {
				log.Printf("Error inserting message: %v", err)
				continue
			}

			// Broadcast to other clients in the session
			message := Message{
				Type:      "MESSAGE",
				SessionID: client.SessionID,
				SenderID:  client.UserID,
				Content:   msg.Content,
				Timestamp: time.Now(),
			}

			h.broadcast <- message
		}
	}
}

func (h *Hub) SendAIMessage(sessionID, content string) {
	message := Message{
		Type:      "MESSAGE",
		SessionID: sessionID,
		SenderID:  "ai",
		Content:   content,
		Timestamp: time.Now(),
	}

	h.broadcast <- message
}

// WebSocketUpgrader configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Configure this properly for production
	},
}
