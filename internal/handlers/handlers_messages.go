package handlers

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/pubsub"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sergioferg/gochat/internal/routing"
	"github.com/sirupsen/logrus"
)

type Message struct {
	ID        uuid.UUID `json:"id"`
	Content   string    `json:"content"`
	SenderID  uuid.UUID `json:"sender_id"`
	ChatID    uuid.UUID `json:"chat_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (api *API) HandlerGetMessages(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Messages []Message `json:"messages"`
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	chatIDStr := r.PathValue("id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid chat ID format", err)
		return
	}

	participants, err := api.DB.GetChatParticipantIDs(r.Context(), chatID)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to verify permissions", err)
		return
	}

	if !slices.Contains(participants, userID) {
		respond.WithError(w, http.StatusForbidden, "You do not have access to this chat", nil)
		return
	}

	var dbMessages []database.Message

	beforeIDStr := r.URL.Query().Get("before_id")

	if beforeIDStr != "" {
		beforeID, err := uuid.Parse(beforeIDStr)
		if err != nil {
			respond.WithError(w, http.StatusBadRequest, "Invalid before_id format", err)
			return
		}

		dbMessages, err = api.DB.GetMessagesBefore(r.Context(), database.GetMessagesBeforeParams{
			ChatID: chatID,
			ID:     beforeID,
		})
	} else {
		dbMessages, err = api.DB.GetRecentMessages(r.Context(), chatID)
	}

	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to fetch messages", err)
		return
	}

	messages := make([]Message, 0, len(dbMessages))

	for _, dbMsg := range dbMessages {
		messages = append(messages, Message{
			ID:        dbMsg.ID,
			Content:   dbMsg.Content,
			SenderID:  dbMsg.SenderID,
			ChatID:    dbMsg.ChatID,
			CreatedAt: dbMsg.CreatedAt,
			UpdatedAt: dbMsg.UpdatedAt,
		})
	}

	respond.WithJSON(w, http.StatusOK, response{
		Messages: messages,
	})
}

func (api *API) HandlerSendMessage(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		ChatID  uuid.UUID `json:"chat_id"`
		Content string    `json:"content"`
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
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

	targetIDs, err := api.DB.GetChatParticipantIDs(r.Context(), params.ChatID)
	if err != nil {
		logrus.Warn("Error getting chat participants:", err)
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	if !slices.Contains(targetIDs, userID) {
		respond.WithError(w, http.StatusForbidden, "You do not have access to this chat", nil)
		return
	}

	if len(targetIDs) == 2 {
		var receiverID uuid.UUID
		if targetIDs[0] == userID {
			receiverID = targetIDs[1]
		} else {
			receiverID = targetIDs[0]
		}

		rel, err := api.DB.GetRelationshipBetweenUsers(r.Context(), database.GetRelationshipBetweenUsersParams{
			ActionUserID: userID,
			TargetUserID: receiverID,
		})

		if err != nil || rel.Status != "accepted" {
			respond.WithError(w, http.StatusForbidden, "Cannot send messages to this user.", nil)
			return
		}

		receiver, err := api.DB.GetUserSingleByID(r.Context(), receiverID)
		if err != nil || receiver.Status == "deleted" {
			respond.WithError(w, http.StatusForbidden, "Cannot send messages to deleted users.", nil)
			return
		}
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

	event := routing.ChatEvent{
		Type:          "new_message",
		ChatID:        params.ChatID,
		SenderID:      userID,
		MessageID:     message.ID,
		Content:       params.Content,
		TargetUserIDs: targetIDs,
	}

	err = pubsub.PublishJSON(api.RMQ.Channel, routing.ChatPrefix, "", event)
	if err != nil {
		logrus.Errorf("Failed to publish message to RabbitMQ: %v", err)
	}

	respond.WithJSON(w, http.StatusCreated, message)
}

func (api *API) HandlerUpdateMessage(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Content string `json:"content"`
	}

	type response struct {
		Message
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
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
		respond.WithError(w, http.StatusBadRequest, "Invalid message ID format", err)
		return
	}

	msgMeta, err := api.DB.GetMessageMetadata(r.Context(), messageID)
	if err != nil {
		respond.WithError(w, http.StatusNotFound, "Message not found", err)
		return
	}

	if userID != msgMeta.SenderID {
		respond.WithError(w, http.StatusForbidden, "You did not send this message", nil)
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
