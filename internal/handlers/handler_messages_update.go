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

func (api *API) HandlerUpdateMessage(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Content string `json:"content"`
	}

	type response struct {
		Message
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid JSON payload", err)
		return
	}

	if strings.TrimSpace(params.Content) == "" {
		respond.WithError(w, http.StatusBadRequest, "Message content cannot be empty", nil)
		return
	}

	messageIDStr := r.PathValue("id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid message ID format", err) // Fixed text
		return
	}

	msgMeta, err := api.DB.GetMessageMetadata(r.Context(), messageID)
	if err != nil {
		respond.WithError(w, http.StatusNotFound, "Message not found", err)
		return
	}

	if userID != msgMeta.SenderID {
		respond.WithError(w, http.StatusUnauthorized, "You did not send this message", nil)
		return
	}

	dbMsg, err := api.DB.UpdateMessage(r.Context(), database.UpdateMessageParams{
		Content: params.Content,
		ID:      messageID,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Couldn't edit message", nil)
		return
	}

	targetIDs, err := api.DB.GetChatParticipantIDs(r.Context(), msgMeta.ChatID)
    if err == nil {
        event := routing.ChatEvent{
            Type:          "message_edited",
            ChatID:        msgMeta.ChatID,
            MessageID:     dbMsg.ID,
            SenderID:      userID,
            Content:       dbMsg.Content,
            TargetUserIDs: targetIDs,
        }
	
        err = pubsub.PublishJSON(api.RMQ.Channel, routing.ChatPrefix, "", event)
        if err != nil {
            logrus.Errorf("Failed to publish edit event to RabbitMQ: %v", err)
        }
    } else {
        logrus.Errorf("Failed to fetch participants for edit event: %v", err)
    }

	respond.WithJSON(w, http.StatusOK, response{
		Message: Message{
			ID:        dbMsg.ID,
			Content:   dbMsg.Content,
			SenderID:  dbMsg.SenderID,
			ChatID:    dbMsg.ChatID,
			CreatedAt: dbMsg.CreatedAt,
			UpdatedAt: dbMsg.UpdatedAt,
		},
	})
}
