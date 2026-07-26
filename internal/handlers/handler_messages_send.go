package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/pubsub"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sergioferg/gochat/internal/routing"
	"github.com/sirupsen/logrus"
)

func (api *API) HandlerSendMessage(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		ChatID  uuid.UUID `json:"chat_id"`
		Content string    `json:"content"`
	}

	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if strings.TrimSpace(params.Content) == "" {
		respond.WithError(w, http.StatusBadRequest, "Message content cannot be empty", nil)
		return
	}

	message, err := api.DB.CreateMessage(r.Context(), database.CreateMessageParams{
		ID:       uuid.Must(uuid.NewV7()),
		Content:  params.Content,
		SenderID: userID,
		ChatID:   params.ChatID,
	})
	if err != nil {
		logrus.Warn("Error creating message:", err)
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	targetIDs, err := api.DB.GetChatParticipantIDs(r.Context(), params.ChatID)
	if err != nil {
		logrus.Warn("Error getting chat participants:", err)
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	event := routing.ChatEvent{
		Type:          "new_message",
		ChatID:        params.ChatID,
		SenderID:      userID,
		Content:       params.Content,
		TargetUserIDs: targetIDs,
	}

	err = pubsub.PublishJSON(api.RMQ.Channel, routing.ChatPrefix, "", event)
	if err != nil {
		logrus.Errorf("Failed to publish message to RabbitMQ: %v", err)
	}

	respond.WithJSON(w, http.StatusCreated, message)
}
