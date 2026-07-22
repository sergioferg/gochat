package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sirupsen/logrus"
)

type User struct {
	ID         uuid.UUID `json:"id"`
	Nickname   string    `json:"nickname"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Email      string    `json:"email"`
	Password   string    `json:"-"`
}

func (api *API) HandlerUserCreate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Nickname string `json:"nickname"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	type response struct {
		User
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Error hashing password", err)
	}

	user, err := api.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.Must(uuid.NewV7()),
		Email:          params.Email,
		Nickname:       params.Nickname,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		if database.IsPgErrorCode(err, "23503") {
			logrus.Warn("Conflict creating user - email exists:", err)
			respond.WithError(w, http.StatusConflict, "A user with this email already exists", err)
			return
		}
		logrus.Warn("Error creating user:", err)
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respond.WithJSON(w, http.StatusCreated, response{
		User: User{
			ID:         user.ID,
			Nickname:   user.Nickname,
			IsVerified: user.IsVerified,
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
			Email:      user.Email,
		},
	})
}
