package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sergioferg/gochat/internal/utils"
	"github.com/sirupsen/logrus"
)

func (api *API) HandlerGitHubLogin(w http.ResponseWriter, r *http.Request) {
	state := auth.GenerateStateOauthCookie(w)

	url := api.GithubOauthCfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (api *API) HandlerGitHubCallback(w http.ResponseWriter, r *http.Request) {
	oauthStateCookie, err := r.Cookie("oauthstate")
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Missing state cookie", err)
		return
	}

	urlState := r.FormValue("state")

	if oauthStateCookie.Value != urlState {
		respond.WithError(w, http.StatusForbidden, "Invalid OAuth state", nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauthstate",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
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
		emailsResp, err := client.Get("https://api.github.com/user/emails")
		if err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to get private emails", err)
			return
		}
		defer emailsResp.Body.Close()

		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}

		if err := json.NewDecoder(emailsResp.Body).Decode(&emails); err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Failed to decode emails", err)
			return
		}

		for _, e := range emails {
			if e.Primary && e.Verified {
				githubUser.Email = e.Email
				break
			}
		}

		if githubUser.Email == "" {
			respond.WithError(w, http.StatusBadRequest, "No verified primary email found on GitHub", nil)
			return
		}
	}

	if valid, err := utils.IsValidAndTrustworthyEmail(githubUser.Email); !valid || err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid email address or unsupported email provider.", err)
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
						RealName:       githubUser.Login,
						BirthDate:      time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
						HashedPassword: nil,
						Status:         "active",
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
				if existingUser.Status == "unverified" {
					err = api.DB.VerifyUser(r.Context(), existingUser.ID)
					if err != nil {
						logrus.Error("Failed to update user status to active during OAuth link:", err)
					}
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
		UserID:    userID,
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

	frontendURL := fmt.Sprintf("%s/oauth-callback#access_token=%s", api.FrontendURL, accessToken)

	http.Redirect(w, r, frontendURL, http.StatusFound)
}
