package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
)

func (api *API) HandlerGitHubLogin(w http.ResponseWriter, r *http.Request) {
	state := auth.GenerateStateOauthCookie(w)

	url := api.GithubOauthCfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (api *API) HandlerGitHubCallback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("oauthstate")
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Missing state cookie", err)
		return
	}

	urlState := r.FormValue("state")

	if cookie.Value != urlState {
		respond.WithError(w, http.StatusForbidden, "Invalid OAuth state", nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "oauthstate",
		Value:  "",
		MaxAge: -1,
	})

	code := r.FormValue("code")
	if code == "" {
		respond.WithError(w, http.StatusBadRequest, "Missing code", nil)
		return
	}

	token, err := api.GithubOauthCfg.Exchange(r.Context(), code)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to exchange token", err)
		return
	}

	client := api.GithubOauthCfg.Client(r.Context(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to get user info", err)
		return
	}
	defer resp.Body.Close()

	var githubUser struct {
		Login string `json:"login"`
		Email string `json:"email"`
		ID    int64  `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to decode user info", err)
		return
	}

	if githubUser.Email == "" {
		respond.WithError(w, http.StatusBadRequest, "Your GitHub email is private. Please make it public or use a password to sign up.", nil)
		return
	}

	providerUserID := strconv.FormatInt(githubUser.ID, 10)

	userID, err := api.DB.GetUserIdFromGithub(r.Context(), providerUserID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {

			// New Github Account

			existingUser, err := api.DB.GetUserByEmail(r.Context(), githubUser.Email)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					user, err := api.DB.CreateUser(r.Context(), database.CreateUserParams{
						ID:             uuid.Must(uuid.NewV7()),
						Email:          githubUser.Email,
						Nickname:       githubUser.Login,
						HashedPassword: nil,
					})
					if err != nil {
						respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
						return
					}

					_, err = api.DB.CreateOAuthGithubAccount(r.Context(), database.CreateOAuthGithubAccountParams{
						ID:             uuid.Must(uuid.NewV7()),
						UserID:         user.ID,
						ProviderUserID: providerUserID,
					})
					if err != nil {
						respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
						return
					}

					userID = user.ID
				} else {
					respond.WithError(w, http.StatusInternalServerError, "Database error", err)
					return
				}
			} else {
				_, err = api.DB.CreateOAuthGithubAccount(r.Context(), database.CreateOAuthGithubAccountParams{
					ID:             uuid.Must(uuid.NewV7()),
					UserID:         existingUser.ID,
					ProviderUserID: providerUserID,
				})
				if err != nil {
					respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
					return
				}

				userID = existingUser.ID
			}

		} else {
			respond.WithError(w, http.StatusInternalServerError, "Failed to check oauth accounts", err)
			return
		}
	} else {
		// Do nothing lmao
	}

	tokenDuration := time.Duration(1 * time.Hour)

	accessToken, err := auth.MakeJWT(userID, api.Secret, tokenDuration)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error generating JWT token", err)
		return
	}

	refreshToken, hashedRefreshToken, err := auth.GenerateAndHashToken()
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "error generating refresh token", err)
		return
	}
	_, err = api.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		TokenHash: hashedRefreshToken,
		UserID:    userID,
	})
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Error saving refresh token to database", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true, // Frontend JS cannot read this!
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60 * 24 * 7, // 7 days
	})

	frontendURL := fmt.Sprintf("%s/oauth-callback#access_token=%s", api.BaseURL, accessToken)

	http.Redirect(w, r, frontendURL, http.StatusFound)
}
