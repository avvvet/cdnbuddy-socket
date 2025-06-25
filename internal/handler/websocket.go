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

var socketService = service.NewSocketService()

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Generate session ID
	sessionID := generateSessionID()
	client := socketService.AddClient(conn, sessionID)
	defer socketService.RemoveClient(conn)

	// Start goroutines for reading and writing
	go writePump(client)
	readPump(client)
}

func readPump(client *service.Client) {
	defer client.GetConn().Close()

	for {
		_, message, err := client.GetConn().ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error: %v", err)
			}
			break
		}

		socketService.HandleMessage(client, message)
	}
}

func writePump(client *service.Client) {
	defer client.GetConn().Close()

	for {
		select {
		case message, ok := <-client.GetSend():
			if !ok {
				client.GetConn().WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.GetConn().WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("❌ Write error: %v", err)
				return
			}
		}
	}
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
