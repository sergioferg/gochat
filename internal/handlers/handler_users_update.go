package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sirupsen/logrus"
)

func (api *API) HandlerUserUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized to do this", nil)
		return
	}

	type parameters struct {
		NewNickname *string `json:"nickname,omitempty"`
		NewPassword *string `json:"password,omitempty"`
		NewEmail    *string `json:"email,omitempty"`
	}

	type response struct {
		User
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if params.NewNickname == nil && params.NewEmail == nil && params.NewPassword == nil {
		respond.WithError(w, http.StatusBadRequest, "No fields provided to update", nil)
		return
	}

	var hashedPassword *string
	if params.NewPassword != nil && *params.NewPassword != "" {
		hash, err := auth.HashPassword(*params.NewPassword)
		if err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Error hashing password", err)
			return
		}
		hashedPassword = &hash
	}

	arg := database.UpdateUserParams{
		ID:             userID,
		Nickname:       params.NewNickname,
		Email:          params.NewEmail,
		HashedPassword: hashedPassword,
	}

	user, err := api.DB.UpdateUser(r.Context(), arg)
	if err != nil {
		if database.IsPgErrorCode(err, "23505") {
			logrus.Warn("Conflict updating user - email exists:", err)
			respond.WithError(w, http.StatusConflict, "A user with this email already exists", err)
			return
		}
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, response{
		User: User{
			ID:        user.ID,
			Nickname:  user.Nickname,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		},
	})
}
