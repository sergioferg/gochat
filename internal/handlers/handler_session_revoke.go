package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerRevokeSession(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := r.PathValue("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid session ID format", err)
		return
	}

	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	err = api.DB.RevokeSession(r.Context(), database.RevokeSessionParams{
		ID:     sessionID,
		UserID: userID,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Could not revoke session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
