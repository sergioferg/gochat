package handlers

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sirupsen/logrus"
)

type Chat struct {
	ID        uuid.UUID `json:"id"`
	Name      *string   `json:"name"`
	IsGroup   bool      `json:"is_group"`
	CreatedAt time.Time `json:"created_at"`
}

func (api *API) HandlerCreateChat(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name           *string     `json:"name"`
		IsGroup        bool        `json:"is_group"`
		ParticipantIDs []uuid.UUID `json:"participant_ids"`
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

	participants := params.ParticipantIDs
	if !slices.Contains(participants, userID) {
		participants = append(participants, userID)
	}

	if !params.IsGroup {
		if len(participants) != 2 {
			respond.WithError(w, http.StatusBadRequest, "1-on-1 chats must have exactly 2 participants", nil)
			return
		}

		var otherUserID uuid.UUID
		if participants[0] == userID {
			otherUserID = participants[1]
		} else {
			otherUserID = participants[0]
		}

		existingChatID, err := api.DB.GetDirectChatBetweenUsers(r.Context(), database.GetDirectChatBetweenUsersParams{
			UserID:   userID,
			UserID_2: otherUserID,
		})

		if err == nil {
			respond.WithJSON(w, http.StatusOK, map[string]any{
				"id":      existingChatID,
				"message": "Chat already exists",
			})
			return
		}
	}

	chatID := uuid.Must(uuid.NewV7())
	chat, err := api.DB.CreateChat(r.Context(), database.CreateChatParams{
		ID:      chatID,
		Name:    params.Name,
		IsGroup: params.IsGroup,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to create chat", err)
		return
	}

	for _, participantID := range participants {
		err = api.DB.AddUserToChat(r.Context(), database.AddUserToChatParams{
			ChatID: chatID,
			UserID: participantID,
		})
		if err != nil {
			logrus.Errorf("Failed to add user %v to chat %v: %v", participantID, chatID, err)
		}
	}

	respond.WithJSON(w, http.StatusCreated, Chat{
		ID:        chat.ID,
		Name:      chat.Name,
		IsGroup:   chat.IsGroup,
		CreatedAt: chat.CreatedAt,
	})
}

type ChatParticipant struct {
	ID       uuid.UUID `json:"id"`
	Nickname string    `json:"nickname"`
	RealName string    `json:"real_name"`
	Status   string    `json:"status"`
}

func (api *API) HandlerGetUserChats(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Chat
		LastReadAt         time.Time         `json:"last_read_at"`
		LastMessageContent *string           `json:"last_message_content"`
		LastMessageID      *uuid.UUID        `json:"last_message_id"`
		Participants       []ChatParticipant `json:"participants"`
		CanSendMessages    bool              `json:"can_send_messages"`
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
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

		dbParticipants, err := api.DB.GetChatParticipantsDetails(r.Context(), dbRow.ChatID)
		participants := make([]ChatParticipant, 0)
		if err == nil {
			for _, p := range dbParticipants {
				participants = append(participants, ChatParticipant{
					ID:       p.ID,
					Nickname: p.Nickname,
					RealName: p.RealName,
					Status:   p.Status,
				})
			}
		}

		canSend := true
		if !dbRow.IsGroup && len(participants) == 2 {
			var receiverID uuid.UUID
			if participants[0].ID == userID {
				receiverID = participants[1].ID
			} else {
				receiverID = participants[0].ID
			}

			rel, err := api.DB.GetRelationshipBetweenUsers(r.Context(), database.GetRelationshipBetweenUsersParams{
				ActionUserID: userID,
				TargetUserID: receiverID,
			})
			if err != nil || rel.Status != "accepted" {
				canSend = false
			}
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
			Participants:       participants,
			CanSendMessages:    canSend,
		})
	}

	respond.WithJSON(w, http.StatusOK, responses)
}
