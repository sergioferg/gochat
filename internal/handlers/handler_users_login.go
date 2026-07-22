package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerUserLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	user, err := api.DB.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respond.WithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}
	if !user.IsVerified {
		respond.WithError(w, http.StatusUnauthorized, "Account not verified", nil)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error checking hash", err)
		return
	}

	if !match {
		respond.WithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}
	// TODO: Move hardcoded JWT token duration (1 hour) to environment variables/configuration.
	tokenDuration := time.Duration(1 * time.Hour)

	accessToken, err := auth.MakeJWT(user.ID, api.Secret, tokenDuration)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error generating JWT token", err)
		return
	}

	refreshToken, hashedRefreshToken, err := auth.GenerateAndHashToken()
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "error generating refresh token", err)
		return
	}
	_, err = api.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		TokenHash: hashedRefreshToken,
		UserID:    user.ID,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error saving refresh token to database", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, response{
		User: User{
			ID:         user.ID,
			Nickname:   user.Nickname,
			IsVerified: user.IsVerified,
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
			Email:      user.Email,
		},
		Token:        accessToken,
		RefreshToken: refreshToken,
	})
}
