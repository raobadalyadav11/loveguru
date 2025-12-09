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

type TypingIndicator struct {
	Type      string    `json:"type"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	IsTyping  bool      `json:"is_typing"`
	Timestamp time.Time `json:"timestamp"`
}

type ReadReceipt struct {
	Type      string    `json:"type"`
	SessionID string    `json:"session_id"`
	MessageID string    `json:"message_id"`
	ReaderID  string    `json:"reader_id"`
	ReadAt    time.Time `json:"read_at"`
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

	// Track typing state
	typingTimer := time.AfterFunc(3*time.Second, func() {
		// Send typing stopped message after 3 seconds of inactivity
		typingIndicator := TypingIndicator{
			Type:      "TYPING_STOPPED",
			SessionID: client.SessionID,
			UserID:    client.UserID,
			IsTyping:  false,
			Timestamp: time.Now(),
		}
		h.broadcastTypingIndicator(typingIndicator)
	})

	for {
		var msg Message
		if err := client.Conn.ReadJSON(&msg); err != nil {
			break
		}

		// Reset typing timer
		typingTimer.Reset(3 * time.Second)

		// Process different message types
		switch msg.Type {
		case "MESSAGE":
			if msg.Content != "" {
				// Store message in database
				messageID, err := h.service.InsertMessageWithID(h.ctx, client.SessionID, "USER", client.UserID, msg.Content)
				if err != nil {
					log.Printf("Error inserting message: %v", err)
					continue
				}

				// Send typing stopped when sending message
				typingIndicator := TypingIndicator{
					Type:      "TYPING_STOPPED",
					SessionID: client.SessionID,
					UserID:    client.UserID,
					IsTyping:  false,
					Timestamp: time.Now(),
				}
				h.broadcastTypingIndicator(typingIndicator)

				// Broadcast message to other clients in the session
				message := Message{
					Type:      "MESSAGE",
					SessionID: client.SessionID,
					SenderID:  client.UserID,
					Content:   msg.Content,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"message_id": messageID,
					},
				}

				h.broadcast <- message
			}

		case "TYPING_STARTED":
			typingIndicator := TypingIndicator{
				Type:      "TYPING_STARTED",
				SessionID: client.SessionID,
				UserID:    client.UserID,
				IsTyping:  true,
				Timestamp: time.Now(),
			}
			h.broadcastTypingIndicator(typingIndicator)

		case "TYPING_STOPPED":
			typingIndicator := TypingIndicator{
				Type:      "TYPING_STOPPED",
				SessionID: client.SessionID,
				UserID:    client.UserID,
				IsTyping:  false,
				Timestamp: time.Now(),
			}
			h.broadcastTypingIndicator(typingIndicator)

		case "READ_RECEIPT":
			if msg.Data != nil {
				if dataMap, ok := msg.Data.(map[string]interface{}); ok {
					if messageID, ok := dataMap["message_id"].(string); ok {
						readReceipt := ReadReceipt{
							Type:      "READ_RECEIPT",
							SessionID: client.SessionID,
							MessageID: messageID,
							ReaderID:  client.UserID,
							ReadAt:    time.Now(),
						}
						h.broadcastReadReceipt(readReceipt)
					}
				}
			}
		}
	}

	// Send typing stopped when connection closes
	typingIndicator := TypingIndicator{
		Type:      "TYPING_STOPPED",
		SessionID: client.SessionID,
		UserID:    client.UserID,
		IsTyping:  false,
		Timestamp: time.Now(),
	}
	h.broadcastTypingIndicator(typingIndicator)
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

func (h *Hub) broadcastTypingIndicator(indicator TypingIndicator) {
	h.clientLock.RLock()
	defer h.clientLock.RUnlock()

	for _, client := range h.clients {
		if client.SessionID == indicator.SessionID && client.UserID != indicator.UserID {
			select {
			case client.Send <- Message{
				Type:      indicator.Type,
				SessionID: indicator.SessionID,
				Timestamp: indicator.Timestamp,
				Data:      indicator,
			}:
			default:
				close(client.Send)
				delete(h.clients, client.ID)
			}
		}
	}
}

func (h *Hub) broadcastReadReceipt(receipt ReadReceipt) {
	h.clientLock.RLock()
	defer h.clientLock.RUnlock()

	for _, client := range h.clients {
		if client.SessionID == receipt.SessionID && client.UserID != receipt.ReaderID {
			select {
			case client.Send <- Message{
				Type:      receipt.Type,
				SessionID: receipt.SessionID,
				Timestamp: receipt.ReadAt,
				Data:      receipt,
			}:
			default:
				close(client.Send)
				delete(h.clients, client.ID)
			}
		}
	}
}

// WebSocketUpgrader configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Configure this properly for production
	},
}

// Enhanced Hub with metrics and better connection management
type EnhancedHub struct {
	*Hub
	metrics           *HubMetrics
	connectionLimit   int
	activeConnections int
	maxConnections    int
}

type HubMetrics struct {
	TotalConnections   int64
	ActiveConnections  int64
	MessagesSent       int64
	MessagesReceived   int64
	Disconnections     int64
	AverageSessionTime time.Duration
	LastConnectionTime time.Time
	mu                 sync.RWMutex
}

func NewEnhancedHub(service *Service) *EnhancedHub {
	return &EnhancedHub{
		Hub:             NewHub(service),
		metrics:         &HubMetrics{},
		connectionLimit: 1000,
		maxConnections:  1000,
	}
}

// GetMetrics returns current hub metrics
func (h *EnhancedHub) GetMetrics() *HubMetrics {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()

	return &HubMetrics{
		TotalConnections:   h.metrics.TotalConnections,
		ActiveConnections:  h.metrics.ActiveConnections,
		MessagesSent:       h.metrics.MessagesSent,
		MessagesReceived:   h.metrics.MessagesReceived,
		Disconnections:     h.metrics.Disconnections,
		AverageSessionTime: h.metrics.AverageSessionTime,
		LastConnectionTime: h.metrics.LastConnectionTime,
	}
}

// registerClient enhanced with metrics
func (h *EnhancedHub) registerClient(client *Client) {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	// Check connection limit
	if len(h.clients) >= h.connectionLimit {
		log.Printf("Connection limit reached, rejecting client %s", client.ID)
		return
	}

	h.clients[client.ID] = client
	h.activeConnections++

	// Update metrics
	h.metrics.mu.Lock()
	h.metrics.TotalConnections++
	h.metrics.ActiveConnections = int64(h.activeConnections)
	h.metrics.LastConnectionTime = time.Now()
	h.metrics.mu.Unlock()

	// Send recent messages to the newly connected client
	go h.sendRecentMessages(client)
}

// unregisterClient enhanced with metrics
func (h *EnhancedHub) unregisterClient(client *Client) {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
		h.activeConnections--

		// Update metrics
		h.metrics.mu.Lock()
		h.metrics.ActiveConnections = int64(h.activeConnections)
		h.metrics.Disconnections++
		h.metrics.mu.Unlock()
	}
}

// broadcastMessage enhanced with metrics
func (h *EnhancedHub) broadcastMessage(message Message) {
	h.clientLock.RLock()
	defer h.clientLock.RUnlock()

	for _, client := range h.clients {
		if client.SessionID == message.SessionID {
			select {
			case client.Send <- message:
				// Update message sent metrics
				h.metrics.mu.Lock()
				h.metrics.MessagesSent++
				h.metrics.mu.Unlock()
			default:
				close(client.Send)
				delete(h.clients, client.ID)
			}
		}
	}
}

// Health check endpoint for monitoring
func (h *EnhancedHub) HealthCheck() map[string]interface{} {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()

	return map[string]interface{}{
		"status":             "healthy",
		"active_connections": h.metrics.ActiveConnections,
		"total_connections":  h.metrics.TotalConnections,
		"max_connections":    h.maxConnections,
		"connection_usage":   float64(h.metrics.ActiveConnections) / float64(h.maxConnections),
		"messages_sent":      h.metrics.MessagesSent,
		"messages_received":  h.metrics.MessagesReceived,
		"disconnections":     h.metrics.Disconnections,
		"last_connection":    h.metrics.LastConnectionTime,
	}
}

// Graceful shutdown
func (h *EnhancedHub) Shutdown(ctx context.Context) error {
	// Close all client connections
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	for _, client := range h.clients {
		close(client.Send)
		client.Conn.Close()
	}

	// Clear clients map
	h.clients = make(map[string]*Client)
	h.activeConnections = 0

	// Wait for shutdown or timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Connection quality monitoring
func (h *EnhancedHub) monitorConnectionQuality() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.checkConnectionQuality()
		}
	}
}

func (h *EnhancedHub) checkConnectionQuality() {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	var slowConnections int
	var inactiveConnections int

	now := time.Now()
	for _, client := range h.clients {
		// Check for slow connections (high latency)
		if h.isSlowConnection(client) {
			slowConnections++
		}

		// Check for inactive connections
		if h.isInactiveConnection(client, now) {
			inactiveConnections++
		}
	}

	// Log metrics
	if slowConnections > 0 || inactiveConnections > 0 {
		log.Printf("Connection quality check: slow=%d, inactive=%d", slowConnections, inactiveConnections)
	}
}

func (h *EnhancedHub) isSlowConnection(client *Client) bool {
	// Simple heuristic: check if write queue is backing up
	return len(client.Send) > 100
}

func (h *EnhancedHub) isInactiveConnection(client *Client, now time.Time) bool {
	// Check if connection has been inactive for more than 5 minutes
	// This would require tracking last activity time per client
	return false // Placeholder - would need client activity tracking
}

// Load balancing for multiple hub instances
type HubManager struct {
	hubs     map[string]*EnhancedHub
	hubIndex int
	mu       sync.RWMutex
	service  *Service
}

func NewHubManager(service *Service) *HubManager {
	return &HubManager{
		hubs:    make(map[string]*EnhancedHub),
		service: service,
	}
}

func (hm *HubManager) GetHub(sessionID string) *EnhancedHub {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// Simple hash-based routing
	hash := len(sessionID) % len(hm.hubs)
	var hub *EnhancedHub
	i := 0
	for _, h := range hm.hubs {
		if i == hash {
			hub = h
			break
		}
		i++
	}

	if hub == nil {
		// Fallback to first available hub
		for _, h := range hm.hubs {
			return h
		}
	}

	return hub
}

func (hm *HubManager) AddHub(id string) *EnhancedHub {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hub := NewEnhancedHub(hm.service)
	hm.hubs[id] = hub

	// Start hub in background
	go hub.Run()
	go hub.monitorConnectionQuality()

	return hub
}

func (hm *HubManager) GetAllMetrics() map[string]*HubMetrics {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	metrics := make(map[string]*HubMetrics)
	for id, hub := range hm.hubs {
		metrics[id] = hub.GetMetrics()
	}

	return metrics
}
