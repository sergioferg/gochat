package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sirupsen/logrus"
)

type UserProfileResponse struct {
	ID                 uuid.UUID `json:"id"`
	Nickname           string    `json:"nickname"`
	RealName           string    `json:"real_name"`
	Status             string    `json:"status"`
	RelationshipStatus string    `json:"relationship_status"`
}

func (api *API) HandlerGetUserProfile(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	targetIDStr := r.PathValue("id")
	targetUserID, err := uuid.Parse(targetIDStr)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	targetUser, err := api.DB.GetUserSingleByID(r.Context(), targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.WithError(w, http.StatusNotFound, "User not found", err)
			return
		}
		respond.WithError(w, http.StatusInternalServerError, "Database error", err)
		return
	}

	relStatus := "none"
	if currentUserID == targetUserID {
		relStatus = "self"
	} else {
		rel, err := api.DB.GetRelationshipBetweenUsers(r.Context(), database.GetRelationshipBetweenUsersParams{
			ActionUserID: currentUserID,
			TargetUserID: targetUserID,
		})
		if err == nil {
			switch rel.Status {
			case "accepted":
				relStatus = "friend"
			case "pending":
				if rel.ActionUserID == currentUserID {
					relStatus = "pending_sent"
				} else {
					relStatus = "pending_received"
				}
			case "blocked":
				if rel.ActionUserID == currentUserID {
					relStatus = "blocked_by_me"
				} else {
					relStatus = "blocked_by_them"
				}
			}
		}
	}

	respond.WithJSON(w, http.StatusOK, UserProfileResponse{
		ID:                 targetUser.ID,
		Nickname:           targetUser.Nickname,
		RealName:           targetUser.RealName,
		Status:             targetUser.Status,
		RelationshipStatus: relStatus,
	})
}

func (api *API) HandlerUnfriendUser(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	targetIDStr := r.PathValue("id")
	targetUserID, err := uuid.Parse(targetIDStr)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	err = api.DB.DeleteRelationshipBetweenUsers(r.Context(), database.DeleteRelationshipBetweenUsersParams{
		ActionUserID: currentUserID,
		TargetUserID: targetUserID,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to unfriend user", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, map[string]string{"message": "User unfriended successfully"})
}

func (api *API) HandlerBlockUser(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	type parameters struct {
		TargetUserID uuid.UUID `json:"target_user_id"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if params.TargetUserID == uuid.Nil || params.TargetUserID == currentUserID {
		respond.WithError(w, http.StatusBadRequest, "Invalid target user ID", nil)
		return
	}

	rel, err := api.DB.GetRelationshipBetweenUsers(r.Context(), database.GetRelationshipBetweenUsersParams{
		ActionUserID: currentUserID,
		TargetUserID: params.TargetUserID,
	})

	if err == nil {
		err = api.DB.UpdateRelationshipStatus(r.Context(), database.UpdateRelationshipStatusParams{
			Status:       "blocked",
			ActionUserID: currentUserID,
			TargetUserID: params.TargetUserID,
			ID:           rel.ID,
		})
	} else {
		_, err = api.DB.CreateUserRelationship(r.Context(), database.CreateUserRelationshipParams{
			ID:           uuid.Must(uuid.NewV7()),
			ActionUserID: currentUserID,
			TargetUserID: params.TargetUserID,
			Status:       "blocked",
		})
	}

	if err != nil {
		logrus.Errorf("Failed to block user %v: %v", params.TargetUserID, err)
		respond.WithError(w, http.StatusInternalServerError, "Failed to block user", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, map[string]string{"message": "User blocked successfully"})
}

func (api *API) HandlerUnblockUser(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	targetIDStr := r.PathValue("id")
	targetUserID, err := uuid.Parse(targetIDStr)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	err = api.DB.DeleteRelationshipBetweenUsers(r.Context(), database.DeleteRelationshipBetweenUsersParams{
		ActionUserID: currentUserID,
		TargetUserID: targetUserID,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to unblock user", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, map[string]string{"message": "User unblocked successfully"})
}

func (api *API) HandlerGetBlockedUsers(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	blockedRows, err := api.DB.GetBlockedUsers(r.Context(), currentUserID)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to fetch blocked users", err)
		return
	}

	if blockedRows == nil {
		blockedRows = []database.GetBlockedUsersRow{}
	}

	respond.WithJSON(w, http.StatusOK, map[string]any{
		"blocked_users": blockedRows,
	})
}
