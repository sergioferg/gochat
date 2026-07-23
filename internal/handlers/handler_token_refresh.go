package handlers

import (
	"net/http"
	"time"

	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Missing bearer token", err)
		return
	}

	user, err := api.DB.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respond.WithError(w, http.StatusUnauthorized, "Invalid/Expired token", err)
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, api.Secret, time.Duration(1*time.Hour))
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error generating JWT token", err)
		return
	}

	type response struct {
		Token string `json:"token"`
	}

	respond.WithJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}
