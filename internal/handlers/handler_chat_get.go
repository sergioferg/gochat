package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerGetUserChats(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Chat
		LastReadAt         time.Time  `json:"last_read_at"`
		LastMessageContent *string    `json:"last_message_content"`
		LastMessageID      *uuid.UUID `json:"last_message_id"`
	}

	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	dbChats, err := api.DB.GetUserChats(r.Context(), userID)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to fetch chats", err)
		return
	}

	if dbChats == nil {
		dbChats = []database.GetUserChatsRow{}
	}

	responses := make([]response, 0, len(dbChats))

	for _, dbRow := range dbChats {

		var lastContent *string
		if dbRow.LastMessageContent != "" {
			content := dbRow.LastMessageContent
			lastContent = &content
		}

		var lastMsgID *uuid.UUID
		if dbRow.LastMessageID != uuid.Nil {
			id := dbRow.LastMessageID
			lastMsgID = &id
		}

		responses = append(responses, response{
			Chat: Chat{
				ID:      dbRow.ChatID,
				Name:    dbRow.ChatName,
				IsGroup: dbRow.IsGroup,
			},
			LastReadAt:         dbRow.LastReadAt,
			LastMessageContent: lastContent,
			LastMessageID:      lastMsgID,
		})
	}

	respond.WithJSON(w, http.StatusOK, responses)
}
