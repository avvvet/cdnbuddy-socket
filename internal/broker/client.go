package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Client wraps NATS connection for socket service
type Client struct {
	conn       *nats.Conn
	publisher  *Publisher
	subscriber *Subscriber
}

// NewClient creates a new NATS messaging client
func NewClient(natsURL string) (*Client, error) {
	// Connect to NATS with options
	opts := []nats.Option{
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			fmt.Printf("NATS disconnected: %v\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %v\n", nc.ConnectedUrl())
		}),
	}

	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	client := &Client{
		conn:       conn,
		publisher:  NewPublisher(conn),
		subscriber: NewSubscriber(conn),
	}

	return client, nil
}

// Publisher returns the publisher instance
func (c *Client) Publisher() *Publisher {
	return c.publisher
}

// Subscriber returns the subscriber instance
func (c *Client) Subscriber() *Subscriber {
	return c.subscriber
}

// Close closes the NATS connection
func (c *Client) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// IsConnected returns true if connected to NATS
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Publisher handles publishing messages to NATS
type Publisher struct {
	conn *nats.Conn
}

// NewPublisher creates a new publisher
func NewPublisher(conn *nats.Conn) *Publisher {
	return &Publisher{conn: conn}
}

// PublishChatMessage publishes a chat message to backend
func (p *Publisher) PublishChatMessage(ctx context.Context, event ChatEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal chat event: %w", err)
	}

	subject := "cdnbuddy.chat"
	return p.conn.Publish(subject, data)
}

// PublishUserConnected notifies backend when user connects
func (p *Publisher) PublishUserConnected(ctx context.Context, userID, sessionID string) error {
	event := UserEvent{
		Type:      "connected",
		UserID:    userID,
		SessionID: sessionID,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal user event: %w", err)
	}

	subject := "user.connected"
	return p.conn.Publish(subject, data)
}

// PublishUserDisconnected notifies backend when user disconnects
func (p *Publisher) PublishUserDisconnected(ctx context.Context, userID, sessionID string) error {
	event := UserEvent{
		Type:      "disconnected",
		UserID:    userID,
		SessionID: sessionID,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal user event: %w", err)
	}

	subject := "user.disconnected"
	return p.conn.Publish(subject, data)
}

// PublishStatusRequest sends a request to API server for CDN status
func (p *Publisher) PublishStatusRequest(ctx context.Context, userID, sessionID string) error {
	event := StatusRequestEvent{
		UserID:    userID,
		SessionID: sessionID,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal status request: %w", err)
	}

	subject := "cdn.status.request"
	return p.conn.Publish(subject, data)
}

// PublishExecuteCommand sends an execute command to API Server
func (p *Publisher) PublishExecuteCommand(ctx context.Context, cmd ExecuteCommand) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal execute command: %w", err)
	}

	subject := "cdnbuddy.execute"
	fmt.Printf("📤 Publishing execute command to %s (PlanID: %s)\n", subject, cmd.PlanID)

	return p.conn.Publish(subject, data)
}

// Subscriber handles subscribing to NATS messages
type Subscriber struct {
	conn *nats.Conn
}

// NewSubscriber creates a new subscriber
func NewSubscriber(conn *nats.Conn) *Subscriber {
	return &Subscriber{conn: conn}
}

// RegisterResponseHandler registers handler for backend responses
func (s *Subscriber) RegisterResponseHandler(handler func(ChatEvent) error) error {
	_, err := s.conn.Subscribe("cdnbuddy.chat.response", func(msg *nats.Msg) {
		var event ChatEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			fmt.Printf("Failed to unmarshal response event: %v\n", err)
			return
		}

		if err := handler(event); err != nil {
			fmt.Printf("Response handler error: %v\n", err)
		}
	})
	return err
}

// RegisterOperationHandler registers handler for operation updates
func (s *Subscriber) RegisterOperationHandler(handler func(OperationEvent) error) error {
	_, err := s.conn.Subscribe("operation.*", func(msg *nats.Msg) {
		var event OperationEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			fmt.Printf("Failed to unmarshal operation event: %v\n", err)
			return
		}

		if err := handler(event); err != nil {
			fmt.Printf("Operation handler error: %v\n", err)
		}
	})
	return err
}

// RegisterServiceHandler registers handler for service events
func (s *Subscriber) RegisterServiceHandler(handler func(ServiceEvent) error) error {
	_, err := s.conn.Subscribe("service.*", func(msg *nats.Msg) {
		var event ServiceEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			fmt.Printf("Failed to unmarshal service event: %v\n", err)
			return
		}

		if err := handler(event); err != nil {
			fmt.Printf("Service handler error: %v\n", err)
		}
	})
	return err
}

// RegisterNotificationHandler registers handler for notifications
func (s *Subscriber) RegisterNotificationHandler(handler func(NotificationEvent) error) error {
	_, err := s.conn.Subscribe("notification.user.*", func(msg *nats.Msg) {
		var event NotificationEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			fmt.Printf("Failed to unmarshal notification event: %v\n", err)
			return
		}

		if err := handler(event); err != nil {
			fmt.Printf("Notification handler error: %v\n", err)
		}
	})
	return err
}

// RegisterStatusResponseHandler registers handler for CDN status responses
func (s *Subscriber) RegisterStatusResponseHandler(handler func(StatusResponseEvent) error) error {
	fmt.Println("📡 Subscribing to cdn.status.response...")
	_, err := s.conn.Subscribe("cdnbuddy.status.response", func(msg *nats.Msg) {
		var event StatusResponseEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			fmt.Printf("Failed to unmarshal status response: %v\n", err)
			return
		}

		if err := handler(event); err != nil {
			fmt.Printf("Status response handler error: %v\n", err)
		}
	})
	return err
}

// RegisterExecutionPlanHandler registers handler for execution plans from API Server
func (s *Subscriber) RegisterExecutionPlanHandler(handler func(ExecutionPlanEvent) error) error {
	fmt.Println("📡 Subscribing to cdnbuddy.execution.plan...")
	_, err := s.conn.Subscribe("cdnbuddy.execution.plan", func(msg *nats.Msg) {
		var event ExecutionPlanEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			fmt.Printf("Failed to unmarshal execution plan: %v\n", err)
			return
		}

		fmt.Printf("📥 NATS: Parsed execution plan for user %s: %s\n", event.UserID, event.Plan.ID)

		if err := handler(event); err != nil {
			fmt.Printf("Execution plan handler error: %v\n", err)
		}
	})
	return err
}
