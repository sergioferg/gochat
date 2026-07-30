package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/pubsub"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sergioferg/gochat/internal/routing"
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

	// We MUST continuously read from the connection.
	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Error("WebSocket error:", err)
			}
			break
		}

		var payload struct {
			Type   string `json:"type"`
			ChatID string `json:"chat_id"`
		}

		if err := json.Unmarshal(rawMsg, &payload); err == nil {
			if payload.Type == "typing" && payload.ChatID != "" {
				logrus.Infof("Received typing event for chat: %s", payload.ChatID)

				chatID, parseErr := uuid.Parse(payload.ChatID)
				if parseErr == nil {
					targetIDs, dbErr := api.DB.GetChatParticipantIDs(r.Context(), chatID)
					if dbErr == nil {
						event := routing.ChatEvent{
							Type:          "typing",
							ChatID:        chatID,
							SenderID:      userID,
							TargetUserIDs: targetIDs,
						}
						if pubErr := pubsub.PublishJSON(api.RMQ.Channel, routing.ChatPrefix, "", event); pubErr != nil {
							logrus.Warn("Failed to publish typing event:", pubErr)
						}
					}
				}
			}
		}
	}
}
