package broker

import (
	"time"

	message "github.com/avvvet/cndbuddy-socket/internal/websocket"
)

// Event types for socket service communication

// ChatEvent represents a chat message from user
type ChatEvent struct {
	Type      string    `json:"type"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// ResponseEvent represents a response from backend to user
type ResponseEvent struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // "text", "markdown", "json"
	Timestamp time.Time `json:"timestamp"`
}

// PlanEvent represents an execution plan from backend
type PlanEvent struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Plan      Plan      `json:"plan"`
	Timestamp time.Time `json:"timestamp"`
}

// Plan represents an execution plan
type Plan struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Steps       []PlanStep `json:"steps"`
	Status      string     `json:"status"` // "pending", "executing", "completed", "failed"
}

// PlanStep represents a single step in execution plan
type PlanStep struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`             // "pending", "executing", "completed", "failed"
	Progress    int    `json:"progress,omitempty"` // 0-100
}

type ExecuteCommand struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	PlanID    string    `json:"plan_id"`
	Timestamp time.Time `json:"timestamp"`
}

// OperationEvent represents operation progress updates
type OperationEvent struct {
	UserID      string    `json:"user_id"`
	OperationID string    `json:"operation_id"`
	Type        string    `json:"type"` // "started", "progress", "completed", "failed"
	Progress    string    `json:"progress,omitempty"`
	Error       string    `json:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ServiceEvent represents service-related events
type ServiceEvent struct {
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"` // "created", "updated", "deleted", "status_changed"
	ServiceID string    `json:"service_id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider,omitempty"`
	Status    string    `json:"status,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NotificationEvent represents general notifications
type NotificationEvent struct {
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"` // "success", "error", "warning", "info"
	Message   string    `json:"message"`
	Title     string    `json:"title,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// UserEvent represents user connection events
type UserEvent struct {
	Type      string    `json:"type"` // "connected", "disconnected"
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ToWebSocketPlan converts broker.Plan to message.Plan
func (p Plan) ToWebSocketPlan() message.Plan {
	// Convert broker steps to websocket steps
	wsSteps := make([]message.PlanStep, len(p.Steps))
	for i, step := range p.Steps {
		wsSteps[i] = message.PlanStep{
			Name:   step.Name,
			Status: step.Status,
		}
	}

	return message.Plan{
		ID:    p.ID,
		Title: p.Title,
		Steps: wsSteps,
	}
}

// StatusRequestEvent is sent to API server to request CDN status
type StatusRequestEvent struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

// StatusResponseEvent is received from API server with CDN status
type StatusResponseEvent struct {
	UserID    string          `json:"user_id"`
	SessionID string          `json:"session_id"`
	Services  []ServiceStatus `json:"services"`
	Timestamp time.Time       `json:"timestamp"`
}

// ServiceStatus represents a CDN service status
type ServiceStatus struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	TestURL  string `json:"test_url"`
	Provider string `json:"provider"`
}

// ExecutionPlanEvent represents an execution plan from API Server
type ExecutionPlanEvent struct {
	UserID    string        `json:"user_id"`
	SessionID string        `json:"session_id"`
	Plan      ExecutionPlan `json:"plan"`
	Timestamp time.Time     `json:"timestamp"`
}

// ExecutionPlan represents a pending execution plan
type ExecutionPlan struct {
	ID                string             `json:"id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	Steps             []string           `json:"steps"`
	EstimatedDuration string             `json:"estimated_duration"`
	Action            string             `json:"action"`
	Parameters        map[string]*string `json:"parameters"`
	CreatedAt         time.Time          `json:"created_at"`
	ExpiresAt         time.Time          `json:"expires_at"`
}
