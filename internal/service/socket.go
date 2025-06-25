package service

import (
	"encoding/json"
	"log"
	"time"

	"github.com/avvvet/cndbuddy-socket/internal/message"
	"github.com/gorilla/websocket"
)

type SocketService struct {
	clients map[*websocket.Conn]*Client
}

type Client struct {
	conn      *websocket.Conn
	sessionID string
	send      chan []byte
}

// Getter methods for Client to access private fields
func (c *Client) GetConn() *websocket.Conn {
	return c.conn
}

func (c *Client) GetSessionID() string {
	return c.sessionID
}

func (c *Client) GetSend() chan []byte {
	return c.send
}

func NewSocketService() *SocketService {
	return &SocketService{
		clients: make(map[*websocket.Conn]*Client),
	}
}

func (s *SocketService) AddClient(conn *websocket.Conn, sessionID string) *Client {
	client := &Client{
		conn:      conn,
		sessionID: sessionID,
		send:      make(chan []byte, 256),
	}

	s.clients[conn] = client
	log.Printf("✅ Client connected: %s", sessionID)
	return client
}

func (s *SocketService) RemoveClient(conn *websocket.Conn) {
	if client, ok := s.clients[conn]; ok {
		log.Printf("❌ Client disconnected: %s", client.sessionID)
		close(client.send)
		delete(s.clients, conn)
	}
}

func (s *SocketService) HandleMessage(client *Client, data []byte) {
	var msg message.IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("❌ Error parsing message: %v", err)
		s.SendError(client, "Invalid message format")
		return
	}

	log.Printf("📥 Received %s from %s", msg.Type, msg.SessionID)

	switch msg.Type {
	case message.TypeChatMessage:
		s.handleChatMessage(client, msg)
	case message.TypeExecutePlan:
		s.handleExecutePlan(client, msg)
	case message.TypeGetStatus:
		s.handleGetStatus(client, msg)
	default:
		s.SendError(client, "Unknown message type")
	}
}

func (s *SocketService) handleChatMessage(client *Client, msg message.IncomingMessage) {
	// Simulate AI response
	response := message.OutgoingMessage{
		Type:      message.TypeAIResponse,
		Content:   "I'll help you with your CDN configuration. Let me create an execution plan.",
		SessionID: msg.SessionID,
		Timestamp: time.Now(),
	}

	s.SendMessage(client, response)

	// Send example execution plan
	time.AfterFunc(1*time.Second, func() {
		plan := message.OutgoingMessage{
			Type: message.TypeExecutionPlan,
			Plan: &message.Plan{
				ID:    "plan_" + time.Now().Format("20060102150405"),
				Title: "Configure CDN for domain",
				Steps: []message.PlanStep{
					{Name: "Validate domain", Status: "pending"},
					{Name: "Configure edge locations", Status: "pending"},
					{Name: "Deploy configuration", Status: "pending"},
				},
			},
			SessionID: msg.SessionID,
			Timestamp: time.Now(),
		}
		s.SendMessage(client, plan)
	})
}

func (s *SocketService) handleExecutePlan(client *Client, msg message.IncomingMessage) {
	// Simulate execution progress
	steps := []string{"Validating domain...", "Configuring edge locations...", "Deploying configuration..."}

	for i, step := range steps {
		time.AfterFunc(time.Duration(i+1)*time.Second, func() {
			progress := message.OutgoingMessage{
				Type:      message.TypeExecutionProgress,
				Message:   step,
				SessionID: msg.SessionID,
				Timestamp: time.Now(),
			}
			s.SendMessage(client, progress)
		})
	}

	// Send completion
	time.AfterFunc(4*time.Second, func() {
		complete := message.OutgoingMessage{
			Type:      message.TypeExecutionComplete,
			Success:   true,
			Message:   "CDN configuration completed successfully!",
			SessionID: msg.SessionID,
			Timestamp: time.Now(),
		}
		s.SendMessage(client, complete)
	})
}

func (s *SocketService) handleGetStatus(client *Client, msg message.IncomingMessage) {
	status := message.OutgoingMessage{
		Type: message.TypeStatusUpdate,
		Status: &message.Status{
			Provider: "CacheFly",
			Domains: []message.Domain{
				{Name: "myapp.com", Status: "active", Regions: 3},
			},
			Metrics: message.Metrics{
				CacheHitRatio:   "94%",
				AvgResponseTime: "45ms",
				TotalRequests:   "1.2M",
			},
		},
		SessionID: msg.SessionID,
		Timestamp: time.Now(),
	}

	s.SendMessage(client, status)
}

func (s *SocketService) SendMessage(client *Client, msg message.OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	select {
	case client.send <- data:
		log.Printf("📤 Sent %s to %s", msg.Type, msg.SessionID)
	default:
		log.Printf("⚠️ Client send buffer full, disconnecting %s", client.sessionID)
		close(client.send)
	}
}

func (s *SocketService) SendError(client *Client, errorMsg string) {
	msg := message.OutgoingMessage{
		Type:      message.TypeError,
		Message:   errorMsg,
		SessionID: client.sessionID,
		Timestamp: time.Now(),
	}
	s.SendMessage(client, msg)
}
