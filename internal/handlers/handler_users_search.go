package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerUsersSearch(w http.ResponseWriter, r *http.Request) {
	type UserSearchResult struct {
		ID       uuid.UUID `json:"id"`
		Nickname string    `json:"nickname"`
		RealName string    `json:"real_name"`
	}

	type response struct {
		Users []UserSearchResult `json:"users"`
	}

	userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized to do this", nil)
		return
	}

	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	if searchQuery == "" {
		respond.WithError(w, http.StatusBadRequest, "Nothing to search for", nil)
		return
	}

	dbUsers, err := api.DB.GetUsersByNickname(r.Context(), database.GetUsersByNicknameParams{
		Nickname: fmt.Sprintf("%%%s%%", searchQuery),
		ID:       userID,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	users := make([]UserSearchResult, 0, len(dbUsers))
	for _, u := range dbUsers {
		users = append(users, UserSearchResult{
			ID:       u.ID,
			Nickname: u.Nickname,
			RealName: u.RealName,
		})
	}

	respond.WithJSON(w, http.StatusOK, response{
		Users: users,
	})
}
