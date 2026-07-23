package handlers

import (
	"context"
	"net/http"

	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/respond"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

func (api *API) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respond.WithError(w, http.StatusUnauthorized, "Invalid/Missing token", err)
			return
		}

		userID, err := auth.ValidateJWT(accessToken, api.Secret)
		if err != nil {
			respond.WithError(w, http.StatusUnauthorized, "Invalid token", err)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
