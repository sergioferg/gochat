package handlers

import (
	"net/http"

	"github.com/sergioferg/gochat/internal/auth"
)

func (api *API) HandlerUserLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		hashedToken := auth.HashToken(cookie.Value)

		_ = api.DB.RevokeSessionByToken(r.Context(), hashedToken)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusNoContent)
}
