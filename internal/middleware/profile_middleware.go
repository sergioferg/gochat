package middleware

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

func (mw *Config) RequireCompletedProfile(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(UserIDContextKey).(uuid.UUID)
		if !ok {
			respond.WithError(w, http.StatusUnauthorized, "Not authorized", nil)
			return
		}

		birthDate, err := mw.DB.GetBirthDateById(r.Context(), userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respond.WithError(w, http.StatusNotFound, "User not found", err)
				return
			}
			respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
			return
		}

		if birthDate == nil || birthDate.IsZero() {
			respond.WithError(w, http.StatusForbidden, "You must complete your profile and provide a valid date of birth before accessing this feature.", nil)
			return
		}

		if calculateAge(*birthDate) < 18 {
			respond.WithError(w, http.StatusForbidden, "You must be at least 18 years old to use GoChat.", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}
