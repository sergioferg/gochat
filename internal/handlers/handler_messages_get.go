package handlers

import (
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
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

	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
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
