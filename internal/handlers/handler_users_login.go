package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerUserLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	type response struct {
		User
		Token string `json:"token"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	cleanEmail := strings.ToLower(strings.TrimSpace(params.Email))

	user, err := api.DB.GetUserByEmail(r.Context(), cleanEmail)
	if err != nil {
		respond.WithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}
	if user.Status == "unverified" {
		respond.WithError(w, http.StatusUnauthorized, "Account not verified", nil)
		return
	}
	if user.HashedPassword == nil {
		respond.WithError(w, http.StatusForbidden, "This email is connected via a social provider (like GitHub). Please log in using that method.", nil)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, *user.HashedPassword)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error checking hash", err)
		return
	}

	if !match {
		respond.WithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}
	// TODO: Move hardcoded JWT token duration (1 hour) to environment variables/configuration.
	tokenDuration := time.Duration(1 * time.Hour)

	accessToken, err := auth.MakeJWT(user.ID, api.Secret, tokenDuration)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error generating JWT token", err)
		return
	}

	refreshToken, hashedRefreshToken, err := auth.GenerateAndHashToken()
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "error generating refresh token", err)
		return
	}

	userAgent := r.UserAgent()

	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	if strings.Contains(ipAddress, ",") {
		ipAddress = strings.Split(ipAddress, ",")[0]
	}
	ipAddress = strings.TrimSpace(ipAddress)
	if host, _, err := net.SplitHostPort(ipAddress); err == nil {
		ipAddress = host
	}
	_, err = api.DB.CreateSession(r.Context(), database.CreateSessionParams{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: hashedRefreshToken,
		UserID:    user.ID,
		UserAgent: userAgent,
		IpAddress: ipAddress,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error saving refresh token to database", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60 * 24 * 60, // 60 days (same as db duration)
	})

	respond.WithJSON(w, http.StatusOK, response{
		User: User{
			ID:        user.ID,
			Nickname:  user.Nickname,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		},
		Token: accessToken,
	})
}
