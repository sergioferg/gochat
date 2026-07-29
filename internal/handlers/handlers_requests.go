package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/respond"
)

type Request struct {
	ID             uuid.UUID `json:"id"`
	SenderID       uuid.UUID `json:"sender_id"`
	SenderNickname string    `json:"sender_nickname,omitempty"`
	SenderRealName string    `json:"sender_real_name,omitempty"`
	TargetUserID   uuid.UUID `json:"target_user_id,omitempty"`
	Status         string    `json:"status,omitempty"`
	InitialMessage *string   `json:"initial_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

func (api *API) HandlerGetRequests(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Requests []Request `json:"requests"`
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	dbRequests, err := api.DB.GetPendingRequestsForUser(r.Context(), userID)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to fetch pending requests", err)
		return
	}

	requests := make([]Request, 0, len(dbRequests))
	for _, req := range dbRequests {
		requests = append(requests, Request{
			ID:             req.ID,
			SenderID:       req.SenderID,
			SenderNickname: req.SenderNickname,
			SenderRealName: req.SenderRealName,
			InitialMessage: req.InitialMessage,
			CreatedAt:      req.CreatedAt,
		})
	}

	respond.WithJSON(w, http.StatusOK, response{
		Requests: requests,
	})
}

func (api *API) HandlerCreateRequest(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		TargetUserID   uuid.UUID `json:"target_user_id"`
		InitialMessage *string   `json:"initial_message"`
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if params.TargetUserID == uuid.Nil {
		respond.WithError(w, http.StatusBadRequest, "target_user_id is required", nil)
		return
	}

	if params.TargetUserID == userID {
		respond.WithError(w, http.StatusBadRequest, "Cannot send a request to yourself", nil)
		return
	}

	if params.InitialMessage != nil {
		trimmedMsg := strings.TrimSpace(*params.InitialMessage)
		if trimmedMsg == "" {
			params.InitialMessage = nil
		} else {
			if len([]rune(trimmedMsg)) > 500 {
				respond.WithError(w, http.StatusBadRequest, "initial_message cannot exceed 500 characters", nil)
				return
			}
			params.InitialMessage = &trimmedMsg
		}
	}

	createdRel, err := api.DB.CreateUserRelationship(r.Context(), database.CreateUserRelationshipParams{
		ID:             uuid.Must(uuid.NewV7()),
		ActionUserID:   userID,
		TargetUserID:   params.TargetUserID,
		Status:         "pending",
		InitialMessage: params.InitialMessage,
	})
	if err != nil {
		if database.IsPgErrorCode(err, "23505") || database.IsPgErrorCode(err, "23514") {
			respond.WithError(w, http.StatusConflict, "A relationship or pending request already exists between these users.", err)
			return
		}
		respond.WithError(w, http.StatusInternalServerError, "Failed to send request", err)
		return
	}

	respond.WithJSON(w, http.StatusCreated, Request{
		ID:             createdRel.ID,
		SenderID:       createdRel.ActionUserID,
		TargetUserID:   createdRel.TargetUserID,
		Status:         createdRel.Status,
		InitialMessage: createdRel.InitialMessage,
		CreatedAt:      createdRel.CreatedAt,
		UpdatedAt:      createdRel.UpdatedAt,
	})
}

func (api *API) HandlerUpdateRequest(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Action string `json:"action"`
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	reqIDStr := r.PathValue("id")
	reqID, err := uuid.Parse(reqIDStr)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid request ID", err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action != "accept" && action != "reject" && action != "block" {
		respond.WithError(w, http.StatusBadRequest, "Invalid action. Must be accept, reject, or block", nil)
		return
	}

	rel, err := api.DB.GetRelationshipByID(r.Context(), reqID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.WithError(w, http.StatusNotFound, "Request not found", err)
			return
		}
		respond.WithError(w, http.StatusInternalServerError, "Database error", err)
		return
	}

	if rel.TargetUserID != userID {
		respond.WithError(w, http.StatusForbidden, "You are not authorized to manage this request", nil)
		return
	}

	if rel.Status != "pending" {
		respond.WithError(w, http.StatusBadRequest, "Request is not in pending status", nil)
		return
	}

	ctx := r.Context()

	switch action {
	case "reject":
		if err := api.DB.DeleteRelationship(ctx, rel.ID); err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to reject request", err)
			return
		}
		respond.WithJSON(w, http.StatusOK, map[string]string{"message": "Request rejected"})

	case "block":
		err = api.DB.UpdateRelationshipStatus(ctx, database.UpdateRelationshipStatusParams{
			Status:       "blocked",
			ActionUserID: userID,           // Current user becomes the blocker
			TargetUserID: rel.ActionUserID, // Original sender becomes the target (blocked)
			ID:           rel.ID,
		})
		if err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to block user", err)
			return
		}
		respond.WithJSON(w, http.StatusOK, map[string]string{"message": "User blocked"})

	case "accept":
		tx, err := api.Pool.Begin(ctx)
		if err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to start transaction", err)
			return
		}
		defer tx.Rollback(ctx)

		qtx := api.DB.WithTx(tx)

		if err := qtx.UpdateRelationshipStatus(ctx, database.UpdateRelationshipStatusParams{
			Status:       "accepted",
			ActionUserID: rel.ActionUserID,
			TargetUserID: rel.TargetUserID,
			ID:           rel.ID,
		}); err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to update relationship status", err)
			return
		}

		chatID := uuid.Must(uuid.NewV7())
		chat, err := qtx.CreateChat(ctx, database.CreateChatParams{
			ID:      chatID,
			Name:    nil,
			IsGroup: false,
		})
		if err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to create chat room", err)
			return
		}

		if err := qtx.AddUserToChat(ctx, database.AddUserToChatParams{
			ChatID: chat.ID,
			UserID: rel.ActionUserID,
		}); err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to add user to chat", err)
			return
		}

		if err := qtx.AddUserToChat(ctx, database.AddUserToChatParams{
			ChatID: chat.ID,
			UserID: rel.TargetUserID,
		}); err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to add target user to chat", err)
			return
		}

		if rel.InitialMessage != nil && strings.TrimSpace(*rel.InitialMessage) != "" {
			msgID := uuid.Must(uuid.NewV7())
			_, err = qtx.CreateMessage(ctx, database.CreateMessageParams{
				ID:       msgID,
				Content:  *rel.InitialMessage,
				SenderID: rel.ActionUserID,
				ChatID:   chat.ID,
			})
			if err != nil {
				respond.WithError(w, http.StatusInternalServerError, "Failed to create initial message", err)
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to commit transaction", err)
			return
		}

		respond.WithJSON(w, http.StatusOK, map[string]any{
			"message": "Request accepted",
			"chat_id": chat.ID,
		})
	}
}
