package handler

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/avvvet/cndbuddy-socket/internal/service"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, check specific origins
		return true
	},
}

var socketService *service.SocketService

// Initialize sets the socket service instance (called from main.go)
func Initialize(service *service.SocketService) {
	socketService = service
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if socketService == nil {
		log.Printf("❌ Socket service not initialized")
		http.Error(w, "Service not available", http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Get UserID from request (multiple options)
	userID := getUserID(r)

	// Generate session ID
	sessionID := generateSessionID()

	// Add client with both UserID and SessionID
	client := socketService.AddClient(conn, userID, sessionID)
	defer socketService.RemoveClient(conn)

	log.Printf("🔗 WebSocket connection established: UserID=%s, SessionID=%s", userID, sessionID)

	// Start goroutines for reading and writing
	go writePump(client)
	readPump(client)
}

func readPump(client *service.Client) {
	defer client.GetConn().Close()

	// Set read deadline and pong handler for keepalive
	client.GetConn().SetReadDeadline(time.Now().Add(60 * time.Second))
	client.GetConn().SetPongHandler(func(string) error {
		client.GetConn().SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.GetConn().ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error for UserID=%s: %v", client.GetUserID(), err)
			}
			break
		}

		// Handle the message via socket service
		socketService.HandleMessage(client, message)
	}
}

func writePump(client *service.Client) {
	// Setup ping ticker for keepalive
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.GetConn().Close()
	}()

	for {
		select {
		case message, ok := <-client.GetSend():
			// Set write deadline
			client.GetConn().SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				// Channel closed, send close message
				client.GetConn().WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the message
			if err := client.GetConn().WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("❌ Write error for UserID=%s: %v", client.GetUserID(), err)
				return
			}

		case <-ticker.C:
			// Send ping for keepalive
			client.GetConn().SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.GetConn().WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("❌ Ping error for UserID=%s: %v", client.GetUserID(), err)
				return
			}
		}
	}
}

// getUserID extracts UserID from request - multiple strategies
func getUserID(r *http.Request) string {
	// Strategy 1: Query parameter
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		return userID
	}

	// Strategy 2: Header (for JWT or API key)
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}

	// Strategy 3: JWT token (TODO: implement JWT parsing)
	if token := r.Header.Get("Authorization"); token != "" {
		// TODO: Parse JWT and extract UserID
		// userID := parseJWTUserID(token)
		// if userID != "" { return userID }
	}

	// Strategy 4: Session cookie (TODO: implement session lookup)
	if cookie, err := r.Cookie("session_id"); err == nil {
		// TODO: Look up UserID from session store
		// userID := lookupUserFromSession(cookie.Value)
		// if userID != "" { return userID }
		_ = cookie // Suppress unused variable warning
	}

	// Fallback: Generate anonymous UserID for development
	return "user_" + randomString(8)
}

func generateSessionID() string {
	return "session_" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
