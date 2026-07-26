package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sirupsen/logrus"
)

func (api *API) HandlerGetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusUnauthorized, "Not authorized", nil)
		return
	}

	dbUsers, err := api.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		logrus.Warn("Database error in /me route:", err)
		respond.WithError(w, http.StatusInternalServerError, "Database error", err)
		return
	}

	if len(dbUsers) == 0 {
		respond.WithError(w, http.StatusNotFound, "User not found", nil)
		return
	}

	type OAuthAccount struct {
		Provider       string `json:"provider"`
		ProviderUserID string `json:"provider_user_id"`
	}

	type UserProfile struct {
		User
		OAuthAccounts []OAuthAccount `json:"oauth_accounts"`
	}

	profile := UserProfile{
		User: User{
			ID:        dbUsers[0].ID,
			Email:     dbUsers[0].Email,
			Nickname:  dbUsers[0].Nickname,
			CreatedAt: dbUsers[0].CreatedAt,
			UpdatedAt: dbUsers[0].UpdatedAt,
		},
		OAuthAccounts: []OAuthAccount{},
	}

	for _, row := range dbUsers {
		profile.OAuthAccounts = append(profile.OAuthAccounts, OAuthAccount{
			Provider:       row.Provider,
			ProviderUserID: row.ProviderUserID,
		})
	}

	respond.WithJSON(w, http.StatusOK, profile)
}
