package handlers

import (
	"net/http"

	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerRevokeToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Missing bearer token", err)
		return
	}

	err = api.DB.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid/Missing token", err)
		return
	}

	type response struct{}

	w.WriteHeader(http.StatusNoContent)
}
