package main

import (
	"log"
	"net/http"
	"os"

	"github.com/avvvet/cndbuddy-socket/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// WebSocket endpoint
	http.HandleFunc("/ws", handler.HandleWebSocket)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Root endpoint with info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"service":"cdnbuddy-socket","status":"running","endpoints":["/ws","/health"]}`))
	})

	log.Printf("🚀 CDNBuddy Socket Service starting on port %s", port)
	log.Printf("📡 WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("❤️  Health check: http://localhost:%s/health", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
