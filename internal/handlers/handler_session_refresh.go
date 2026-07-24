package handlers

import (
	"net/http"
	"time"

	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerRefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		respond.WithError(w, http.StatusUnauthorized, "Missing refresh token cookie", err)
		return
	}
	rawToken := cookie.Value
	hashedToken := auth.HashToken(rawToken)

	user, err := api.DB.GetUserFromSession(r.Context(), hashedToken)
	if err != nil {
		respond.WithError(w, http.StatusUnauthorized, "Invalid/Expired token", err)
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, api.Secret, time.Duration(1*time.Hour))
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error generating JWT token", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}
