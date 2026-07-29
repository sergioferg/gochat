package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerGetSessions(w http.ResponseWriter, r *http.Request) {
	type SessionResponse struct {
		ID        uuid.UUID `json:"id"`
		IsCurrent bool      `json:"is_current"`
		CreatedAt time.Time `json:"created_at"`
		UserAgent string    `json:"user_agent"`
		IpAddress string    `json:"ip_address"`
	}

	type response struct {
		Sessions []SessionResponse `json:"sessions"`
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	var currentHashedToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		currentHashedToken = auth.HashToken(cookie.Value)
	}

	dbSessions, err := api.DB.GetUserSessions(r.Context(), userID)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Could not fetch sessions", err)
		return
	}

	sessions := make([]SessionResponse, 0, len(dbSessions))
	for _, t := range dbSessions {
		sessions = append(sessions, SessionResponse{
			ID:        t.ID,
			IsCurrent: currentHashedToken != "" && t.TokenHash == currentHashedToken,
			CreatedAt: t.CreatedAt,
			UserAgent: t.UserAgent,
			IpAddress: t.IpAddress,
		})
	}

	respond.WithJSON(w, http.StatusOK, response{
		Sessions: sessions,
	})
}

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

func (api *API) HandlerRevokeSession(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := r.PathValue("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid session ID format", err)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	err = api.DB.RevokeSession(r.Context(), database.RevokeSessionParams{
		ID:     sessionID,
		UserID: userID,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Could not revoke session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
