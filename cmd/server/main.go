package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/avvvet/cndbuddy-socket/internal/broker"
	"github.com/avvvet/cndbuddy-socket/internal/handler"
	"github.com/avvvet/cndbuddy-socket/internal/service"
)

func main() {
	// Load configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	log.Printf("🚀 Starting CDNBuddy Socket Service...")
	log.Printf("📡 Port: %s", port)
	log.Printf("🔗 NATS URL: %s", natsURL)

	// Initialize NATS messaging client
	log.Printf("📡 Connecting to NATS...")
	msgClient, err := broker.NewClient(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer msgClient.Close()
	log.Printf("✅ NATS connected")

	// Initialize socket service with NATS
	socketService := service.NewSocketService(msgClient)

	// Setup NATS event handlers for backend communication
	setupNATSHandlers(socketService, msgClient)

	// Initialize WebSocket handler with socket service
	handler.Initialize(socketService)

	// WebSocket endpoint
	http.HandleFunc("/ws", handler.HandleWebSocket)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
            "status": "healthy",
            "service": "cdnbuddy-socket",
            "nats": "connected"
        }`))
	})

	// Root endpoint with info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
            "service": "cdnbuddy-socket",
            "status": "running",
            "endpoints": ["/ws", "/health"],
            "nats": "connected"
        }`))
	})

	log.Printf("🌟 CDNBuddy Socket Service Configuration:")
	log.Printf("   📡 WebSocket: ws://localhost:%s/ws", port)
	log.Printf("   ❤️  Health: http://localhost:%s/health", port)
	log.Printf("   📨 NATS: Connected and ready")
	log.Printf("   🎯 Ready for chat messages")

	// Setup graceful shutdown
	go func() {
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("🛑 Shutting down Socket Service...")
	log.Printf("✅ Socket Service exited gracefully")
}

// setupNATSHandlers configures event handlers for backend communication
func setupNATSHandlers(socketService *service.SocketService, msgClient *broker.Client) {
	subscriber := msgClient.Subscriber()

	// Handle text responses from backend
	err := subscriber.RegisterResponseHandler(func(event broker.ChatEvent) error {
		log.Printf("💬 Response received for user %s: %s", event.UserID, event.Message)

		// Send to connected client via socket service
		return socketService.SendResponse(event)
	})
	if err != nil {
		log.Printf("❌ Failed to register response handler: %v", err)
	}

	// Handle execution plans from backend
	err = subscriber.RegisterPlanHandler(func(event broker.PlanEvent) error {
		log.Printf("📋 Execution Plan received for user %s: %s", event.UserID, event.Plan.Title)

		// Send execution plan to connected client
		wsPlan := event.Plan.ToWebSocketPlan()
		return socketService.SendPlan(event.UserID, event.SessionID, wsPlan)
	})
	if err != nil {
		log.Printf("❌ Failed to register plan handler: %v", err)
	}

	// Handle operation progress updates from backend
	err = subscriber.RegisterOperationHandler(func(event broker.OperationEvent) error {
		log.Printf("⚙️ Operation Update: %s - %s", event.Type, event.OperationID)

		// Send operation update to client
		return socketService.SendOperationUpdate(event.UserID, event.Type, event.Progress)
	})
	if err != nil {
		log.Printf("❌ Failed to register operation handler: %v", err)
	}

	// Handle service events from backend
	err = subscriber.RegisterServiceHandler(func(event broker.ServiceEvent) error {
		log.Printf("📢 Service Event: %s - %s", event.Type, event.Name)

		// Send service update to client
		return socketService.SendServiceUpdate(event.UserID, event.Type, event.Name, event.Provider)
	})
	if err != nil {
		log.Printf("❌ Failed to register service handler: %v", err)
	}

	// Handle notification events from backend
	err = subscriber.RegisterNotificationHandler(func(event broker.NotificationEvent) error {
		log.Printf("🔔 Notification: %s for user %s", event.Message, event.UserID)

		// Send notification to client
		return socketService.SendNotification(event.UserID, event.Type, event.Message)
	})
	if err != nil {
		log.Printf("❌ Failed to register notification handler: %v", err)
	}

	log.Printf("✅ NATS event handlers configured")
}
