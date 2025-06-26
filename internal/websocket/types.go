package message

import "time"

// Incoming message types (from frontend)
type IncomingMessage struct {
	Type      string    `json:"type"`
	Message   string    `json:"message,omitempty"`
	PlanID    string    `json:"plan_id,omitempty"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// Outgoing message types (to frontend)
type OutgoingMessage struct {
	Type      string    `json:"type"`
	Content   string    `json:"content,omitempty"`
	Plan      *Plan     `json:"plan,omitempty"`
	Status    *Status   `json:"status,omitempty"`
	SessionID string    `json:"session_id"`
	Success   bool      `json:"success,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Execution Plan structure
type Plan struct {
	ID    string     `json:"id"`
	Title string     `json:"title"`
	Steps []PlanStep `json:"steps"`
}

type PlanStep struct {
	Name   string `json:"name"`
	Status string `json:"status"` // pending, running, completed, error
}

// CDN Status structure
type Status struct {
	Provider string   `json:"provider"`
	Domains  []Domain `json:"domains"`
	Metrics  Metrics  `json:"metrics"`
}

type Domain struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // active, configuring, error
	Regions int    `json:"regions"`
}

type Metrics struct {
	CacheHitRatio   string `json:"cache_hit_ratio"`
	AvgResponseTime string `json:"avg_response_time"`
	TotalRequests   string `json:"total_requests"`
}

// Message type constants
const (
	// Incoming types
	TypeChatMessage = "chat_message"
	TypeExecutePlan = "execute_plan"
	TypeGetStatus   = "get_status"

	// Outgoing types
	TypeAIResponse        = "ai_response"
	TypeExecutionPlan     = "execution_plan"
	TypeStatusUpdate      = "status_update"
	TypeExecutionProgress = "execution_progress"
	TypeExecutionComplete = "execution_complete"
	TypeError             = "error"
)
