package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerGetSessions(w http.ResponseWriter, r *http.Request) {
	type SessionResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UserAgent string    `json:"user_agent"`
		IpAddress string    `json:"ip_address"`
	}

	type response struct {
		Sessions []SessionResponse `json:"sessions"`
	}

	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
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
			CreatedAt: t.CreatedAt,
			UserAgent: t.UserAgent,
			IpAddress: t.IpAddress,
		})
	}

	respond.WithJSON(w, http.StatusOK, response{
		Sessions: sessions,
	})
}
