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
		var accessToken string

		accessToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			accessToken = r.URL.Query().Get("token")

			if accessToken == "" {
				respond.WithError(w, http.StatusUnauthorized, "User not logged in", nil)
				return
			}
		}

		userID, err := auth.ValidateJWT(accessToken, api.Secret)
		if err != nil {
			respond.WithError(w, http.StatusUnauthorized, "Invalid token", nil)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
