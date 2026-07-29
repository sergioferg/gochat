package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sergioferg/gochat/internal/ws"
	"github.com/sirupsen/logrus"
)

// Upgrader config
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Upgrades the connection and registers the client
func (api *API) HandlerWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	// Upgrade the HTTP connection to a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Info("Failed to upgrade websocket:", err)
		return
	}

	// Create the client wrapper and register it with the Manager
	client := &ws.Client{
		Conn:   conn,
		UserID: userID,
	}
	api.WSManager.AddClient(client)

	// Ensure cleanup happens when the function exits
	defer func() {
		api.WSManager.RemoveClient(client)
		conn.Close()
	}()

	// We MUST continuously read from the connection. Even if the client only
	// receives messages and never sends them, reading processes internal ping/pong
	// frames and detects when the user closes their laptop or loses internet.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Error("WebSocket error:", err)
			}
			break
		}

		// Note: In this architecture, clients send messages via standard POST /api/messages,
		// so we just ignore any incoming WS data here. We strictly use WS for pushing down.
	}
}
