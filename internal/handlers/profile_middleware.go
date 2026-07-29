package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sergioferg/gochat/internal/respond"
)

func calculateAge(dob time.Time) int {
	now := time.Now().UTC()
	age := now.Year() - dob.Year()
	if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
		age--
	}
	return age
}

type profileIncompleteResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (api *API) RequireCompletedProfile(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
		if !ok {
			respond.WithError(w, http.StatusUnauthorized, "Not authorized", nil)
			return
		}

		var birthDate *time.Time
		err := api.Pool.QueryRow(r.Context(), "SELECT birth_date FROM users WHERE id = $1", userID).Scan(&birthDate)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respond.WithError(w, http.StatusNotFound, "User not found", err)
				return
			}
			respond.WithError(w, http.StatusInternalServerError, "Database error", err)
			return
		}

		if birthDate == nil || birthDate.IsZero() {
			respond.WithJSON(w, http.StatusForbidden, profileIncompleteResponse{
				Error:   "profile_incomplete",
				Message: "You must complete your profile and provide a valid date of birth before accessing this feature.",
			})
			return
		}

		if calculateAge(*birthDate) < 18 {
			respond.WithJSON(w, http.StatusForbidden, profileIncompleteResponse{
				Error:   "underage",
				Message: "You must be at least 18 years old to use GoChat.",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
