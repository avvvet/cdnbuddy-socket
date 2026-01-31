package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/avvvet/cndbuddy-socket/internal/broker"
	message "github.com/avvvet/cndbuddy-socket/internal/websocket"
	"github.com/gorilla/websocket"
)

type SocketService struct {
	clients   map[*websocket.Conn]*Client
	userIndex map[string]*Client // Index clients by UserID for quick lookup

	brokerClient *broker.Client
	publisher    *broker.Publisher
	subscriber   *broker.Subscriber
}

type Client struct {
	conn      *websocket.Conn
	userID    string // Add UserID for NATS communication
	sessionID string
	send      chan []byte
}

// Getter methods for Client to access private fields
func (c *Client) GetConn() *websocket.Conn {
	return c.conn
}

func (c *Client) GetUserID() string {
	return c.userID
}

func (c *Client) GetSessionID() string {
	return c.sessionID
}

func (c *Client) GetSend() chan []byte {
	return c.send
}

// NewSocketService creates a new socket service with NATS broker
func NewSocketService(brokerClient *broker.Client) *SocketService {
	return &SocketService{
		clients:      make(map[*websocket.Conn]*Client),
		userIndex:    make(map[string]*Client),
		brokerClient: brokerClient,
		publisher:    brokerClient.Publisher(),
		subscriber:   brokerClient.Subscriber(),
	}
}

func (s *SocketService) AddClient(conn *websocket.Conn, userID, sessionID string) *Client {
	client := &Client{
		conn:      conn,
		userID:    userID,
		sessionID: sessionID,
		send:      make(chan []byte, 256),
	}

	s.clients[conn] = client
	s.userIndex[userID] = client // Index by UserID for NATS responses

	log.Printf("✅ Client connected: UserID=%s, SessionID=%s", userID, sessionID)

	// Notify backend that user connected
	ctx := context.Background()
	if err := s.publisher.PublishUserConnected(ctx, userID, sessionID); err != nil {
		log.Printf("❌ Failed to publish user connected event: %v", err)
	}

	return client
}

func (s *SocketService) RemoveClient(conn *websocket.Conn) {
	if client, ok := s.clients[conn]; ok {
		log.Printf("❌ Client disconnected: UserID=%s, SessionID=%s", client.userID, client.sessionID)

		// Notify backend that user disconnected
		ctx := context.Background()
		if err := s.publisher.PublishUserDisconnected(ctx, client.userID, client.sessionID); err != nil {
			log.Printf("❌ Failed to publish user disconnected event: %v", err)
		}

		// Clean up
		close(client.send)
		delete(s.clients, conn)
		delete(s.userIndex, client.userID)
	}
}

func (s *SocketService) HandleMessage(client *Client, data []byte) {
	var msg message.IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("❌ Error parsing message: %v", err)
		s.SendError(client, "Invalid message format")
		return
	}

	log.Printf("📥 Received %s from UserID=%s, SessionID=%s", msg.Type, client.userID, msg.SessionID)

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
	// Convert to NATS event inline (until broker.FromIncomingMessage is created)
	chatEvent := broker.ChatEvent{
		UserID:    client.userID,
		SessionID: msg.SessionID,
		Message:   msg.Message,
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	if err := s.publisher.PublishChatMessage(ctx, chatEvent); err != nil {
		log.Printf("❌ Failed to publish chat message: %v", err)
		s.SendError(client, "Failed to process your message. Please try again.")
		return
	}

	// Send immediate acknowledgment
	ackResponse := message.OutgoingMessage{
		Type:      message.TypeAIResponse,
		Content:   "🤖 Processing your request...",
		SessionID: msg.SessionID,
		Timestamp: time.Now(),
	}
	s.SendMessage(client, ackResponse)
}

// handleExecutePlan forwards execution request to API Server via NATS
func (s *SocketService) handleExecutePlan(client *Client, msg message.IncomingMessage) {
	if msg.PlanID == "" {
		log.Printf("❌ Execute plan failed: plan_id required")
		s.SendError(client, "Plan ID is required")
		return
	}

	log.Printf("🚀 Execute plan request: UserID=%s, PlanID=%s", client.userID, msg.PlanID)

	// Forward to API Server via NATS
	ctx := context.Background()
	executeCmd := broker.ExecuteCommand{
		UserID:    client.userID,
		SessionID: msg.SessionID,
		PlanID:    msg.PlanID,
		Timestamp: time.Now(),
	}

	if err := s.publisher.PublishExecuteCommand(ctx, executeCmd); err != nil {
		log.Printf("❌ Failed to publish execute command: %v", err)
		s.SendError(client, "Failed to start execution. Please try again.")
		return
	}

	log.Printf("✅ Execute command sent to API Server: PlanID=%s", msg.PlanID)

	// API Server will send back execution_progress and execution_complete via NATS
}

// Keep existing simulation for get status
func (s *SocketService) handleGetStatusOld(client *Client, msg message.IncomingMessage) {
	// TODO: Send to backend via NATS instead of simulation
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

func (s *SocketService) handleGetStatus(client *Client, _ message.IncomingMessage) {
	log.Printf("📡 Requesting CDN status for UserID=%s", client.userID)

	// Send request to API server via NATS
	ctx := context.Background()
	err := s.publisher.PublishStatusRequest(ctx, client.userID, client.sessionID)
	if err != nil {
		log.Printf("❌ Failed to request status: %v", err)
		s.SendError(client, "Failed to fetch CDN status. Please try again.")
		return
	}

	// Response will come back via NATS subscriber
	log.Printf("✅ Status request sent for UserID=%s", client.userID)
}

func (s *SocketService) SendMessage(client *Client, msg message.OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	select {
	case client.send <- data:
		log.Printf("📤 Sent %s to UserID=%s, SessionID=%s", msg.Type, client.userID, msg.SessionID)
	default:
		log.Printf("⚠️ Client send buffer full, disconnecting UserID=%s", client.userID)
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

// NATS Response Methods (called from main.go NATS handlers)

// SendResponse sends a text response from backend to user
func (s *SocketService) SendResponse(event broker.ChatEvent) error {
	client := s.findClientByUser(event.UserID)
	if client == nil {
		log.Printf("⚠️ Client not found for UserID=%s", event.UserID)
		return fmt.Errorf("client not found for UserID=%s", event.UserID)
	}

	response := message.OutgoingMessage{
		Type:      event.Type,
		Message:   event.Message,
		SessionID: event.SessionID,
		Timestamp: time.Now(),
	}

	s.SendMessage(client, response)
	return nil
}

// SendExecutionPlan sends an execution plan from API Server to the connected user
func (s *SocketService) SendExecutionPlan(event broker.ExecutionPlanEvent) error {
	client := s.findClientByUser(event.UserID)
	if client == nil {
		log.Printf("⚠️ Client not found for UserID=%s", event.UserID)
		return fmt.Errorf("client not found for UserID=%s", event.UserID)
	}

	// Convert broker.ExecutionPlan to message.Plan format for WebSocket
	planSteps := make([]message.PlanStep, len(event.Plan.Steps))
	for i, step := range event.Plan.Steps {
		planSteps[i] = message.PlanStep{
			Name:   step,
			Status: "pending",
		}
	}

	wsPlan := message.Plan{
		ID:          event.Plan.ID,
		Title:       event.Plan.Title,
		Description: event.Plan.Description,
		Steps:       planSteps,
	}

	planMessage := message.OutgoingMessage{
		Type:      message.TypeExecutionPlan,
		Plan:      &wsPlan,
		SessionID: event.SessionID,
		Timestamp: time.Now(),
	}

	s.SendMessage(client, planMessage)
	log.Printf("✅ Execution plan sent to UserID=%s (Plan: %s)", event.UserID, event.Plan.ID)
	return nil
}

// SendOperationUpdate sends operation progress from backend to user
func (s *SocketService) SendOperationUpdate(userID, operationType, progress string) error {
	client := s.findClientByUser(userID)
	if client == nil {
		log.Printf("⚠️ Client not found for UserID=%s", userID)
		return fmt.Errorf("⚠️ Client not found for UserID=%s", userID)
	}

	msgType := message.TypeExecutionProgress
	if operationType == "completed" {
		msgType = message.TypeExecutionComplete
	} else if operationType == "failed" {
		msgType = message.TypeError
	}

	update := message.OutgoingMessage{
		Type:      msgType,
		Message:   progress,
		Success:   operationType == "completed",
		SessionID: client.sessionID, // Use client's sessionID
		Timestamp: time.Now(),
	}

	s.SendMessage(client, update)
	return nil
}

// SendServiceUpdate sends service events from backend to user
func (s *SocketService) SendServiceUpdate(userID, eventType, serviceName, provider string) error {
	client := s.findClientByUser(userID)
	if client == nil {
		log.Printf("⚠️ Client not found for UserID=%s", userID)
		return fmt.Errorf("⚠️ Client not found for UserID=%s", userID)
	}

	var content string
	switch eventType {
	case "created":
		content = fmt.Sprintf("✅ Service '%s' created successfully with %s", serviceName, provider)
	case "updated":
		content = fmt.Sprintf("🔄 Service '%s' updated successfully", serviceName)
	case "deleted":
		content = fmt.Sprintf("🗑️ Service '%s' deleted successfully", serviceName)
	default:
		content = fmt.Sprintf("📢 Service '%s' %s", serviceName, eventType)
	}

	response := message.OutgoingMessage{
		Type:      message.TypeAIResponse,
		Content:   content,
		SessionID: client.sessionID,
		Timestamp: time.Now(),
	}

	s.SendMessage(client, response)
	return nil
}

// SendNotification sends general notifications from backend to user
func (s *SocketService) SendNotification(userID, notificationType, notificationMessage string) error {
	client := s.findClientByUser(userID)
	if client == nil {
		log.Printf("⚠️ Client not found for UserID=%s", userID)
		return fmt.Errorf("⚠️ Client not found for UserID=%s", userID)

	}

	// Choose appropriate emoji based on notification type
	var prefix string
	switch notificationType {
	case "success":
		prefix = "✅"
	case "error":
		prefix = "❌"
	case "warning":
		prefix = "⚠️"
	case "info":
		prefix = "ℹ️"
	default:
		prefix = "🔔"
	}

	response := message.OutgoingMessage{
		Type:      message.TypeAIResponse,
		Content:   prefix + " " + notificationMessage,
		SessionID: client.sessionID,
		Timestamp: time.Now(),
	}

	s.SendMessage(client, response)
	return nil
}

// SendStatusUpdate sends CDN status to a specific user
func (s *SocketService) SendStatusUpdate(event broker.StatusResponseEvent) error {
	client := s.findClientByUser(event.UserID)
	if client == nil {
		log.Printf("⚠️ Client not found for UserID=%s", event.UserID)
		return fmt.Errorf("client not found for UserID=%s", event.UserID)
	}

	// Convert to frontend message format
	domains := make([]message.Domain, 0, len(event.Services))
	for _, svc := range event.Services {
		domains = append(domains, message.Domain{
			Name:   svc.Name,
			Status: svc.Status,
		})
	}

	status := message.OutgoingMessage{
		Type: message.TypeStatusUpdate,
		Status: &message.Status{
			Provider: "CacheFly",
			Domains:  domains,
			Metrics: message.Metrics{
				CacheHitRatio:   "N/A",
				AvgResponseTime: "N/A",
				TotalRequests:   "N/A",
			},
		},
		SessionID: event.SessionID,
		Timestamp: time.Now(),
	}

	s.SendMessage(client, status)
	log.Printf("✅ Status update sent to UserID=%s (%d services)", event.UserID, len(event.Services))
	return nil
}

// Helper method to find client by UserID
func (s *SocketService) findClientByUser(userID string) *Client {
	if client, exists := s.userIndex[userID]; exists {
		return client
	}
	return nil
}
