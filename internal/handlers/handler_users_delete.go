package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerUserDelete(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	ctx := r.Context()

	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized to do this", nil)
		return
	}

	tx, err := api.Pool.Begin(ctx)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to start transaction", err)
		return
	}
	defer tx.Rollback(ctx)

	qtx := api.DB.WithTx(tx)

	if err := qtx.DeleteUserRefreshTokens(ctx, userID); err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to delete tokens", err)
		return
	}

	if err := qtx.AnonymizeUser(ctx, userID); err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to delete user", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to commit deletion", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, map[string]string{"message": "Account deleted"})

}
