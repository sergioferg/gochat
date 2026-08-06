package handlers

import (
	"crypto/rand"
	"database/sql"
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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/mailer"
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sergioferg/gochat/internal/utils"
	"github.com/sirupsen/logrus"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	Nickname    string    `json:"nickname"`
	RealName    string    `json:"real_name"`
	DateOfBirth string    `json:"date_of_birth"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	Password    string    `json:"-"`
}

func parseDateOfBirth(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("date_of_birth is required")
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func formatDateOfBirth(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func calculateAge(dob time.Time) int {
	now := time.Now().UTC()
	age := now.Year() - dob.Year()
	if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
		age--
	}
	return age
}

func (api *API) HandlerUserCreate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Nickname    string `json:"nickname"`
		RealName    string `json:"real_name"`
		DateOfBirth string `json:"date_of_birth"`
		Password    string `json:"password"`
		Email       string `json:"email"`
	}
	type response struct {
		User
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if strings.TrimSpace(params.RealName) == "" {
		respond.WithError(w, http.StatusBadRequest, "real_name is required", nil)
		return
	}

	dob, err := parseDateOfBirth(params.DateOfBirth)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid or missing date_of_birth format. Use YYYY-MM-DD", err)
		return
	}
	if calculateAge(dob) < 18 {
		respond.WithError(w, http.StatusBadRequest, "You must be at least 18 years old to use GoChat.", nil)
		return
	}

	if valid, err := utils.IsValidAndTrustworthyEmail(params.Email); !valid || err != nil {
		respond.WithError(w, http.StatusBadRequest, "Invalid email address or unsupported email provider.", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Error hashing password", err)
		return
	}

	user, err := api.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.Must(uuid.NewV7()),
		Email:          params.Email,
		Nickname:       params.Nickname,
		RealName:       params.RealName,
		BirthDate:      &dob,
		HashedPassword: &hashedPassword,
	})
	if err != nil {
		if database.IsPgErrorCode(err, "23505") {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.ConstraintName == "users_nickname_key" {
					respond.WithError(w, http.StatusConflict, "Nickname is already taken", err)
					return
				}
				if pgErr.ConstraintName == "users_email_key" {
					respond.WithError(w, http.StatusConflict, "Email is already registered", err)
					return
				}
			}
			logrus.Warn("Conflict creating user - email or nickname exists:", err)
			respond.WithError(w, http.StatusConflict, "A user with this email or nickname already exists", err)
			return
		}
		logrus.Warn("Error creating user:", err)
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	rawToken, hashedToken, err := auth.GenerateAndHashToken()
	if err != nil {
		logrus.Warn("Error generating and hashing token:", err)
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	_, err = api.DB.CreateVerificationToken(r.Context(), database.CreateVerificationTokenParams{
		TokenHash: hashedToken,
		UserID:    user.ID,
	})
	if err != nil {
		logrus.Warn("Error creating verification token:", err)
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	// TODO: Replace unmanaged goroutine with a job queue or worker pool for reliable asynchronous email delivery and error retries.
	go func(email, nick, token string) {
		url := fmt.Sprintf("%s/verify-email?token=%s", api.FrontendURL, token)
		_ = mailer.SendResendEmail(email, nick, url, api.ResendApiKey)
	}(user.Email, user.Nickname, rawToken)

	respond.WithJSON(w, http.StatusCreated, response{
		User: User{
			ID:          user.ID,
			Nickname:    user.Nickname,
			RealName:    user.RealName,
			DateOfBirth: formatDateOfBirth(user.BirthDate),
			Status:      user.Status,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
		},
	})
}

func (api *API) HandlerUserDelete(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	ctx := r.Context()

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized", nil)
		return
	}

	tx, err := api.Pool.Begin(ctx)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to start transaction", err)
		return
	}
	defer tx.Rollback(ctx)

	qtx := api.DB.WithTx(tx)

	if err := qtx.DeleteUserSessions(ctx, userID); err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to delete tokens", err)
		return
	}

	if err := qtx.AnonymizeUser(ctx, userID); err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to delete user", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Failed to commit deletion", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, map[string]string{"message": "Account deleted"})
}

func (api *API) HandlerGetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusUnauthorized, "Not authorized", nil)
		return
	}

	// TEMPORARY DEBUG LOG
	logrus.Infof("GetMe accessed with user ID: %v", userID)

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
			ID:          dbUsers[0].ID,
			Email:       dbUsers[0].Email,
			Nickname:    dbUsers[0].Nickname,
			RealName:    dbUsers[0].RealName,
			DateOfBirth: formatDateOfBirth(dbUsers[0].BirthDate),
			CreatedAt:   dbUsers[0].CreatedAt,
			UpdatedAt:   dbUsers[0].UpdatedAt,
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
	if user.Status == "deactivated" || user.Status == "deleted" {
		respond.WithError(w, http.StatusForbidden, "Account is "+user.Status, nil)
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

	auth.SetAuthCookie(w, refreshToken)

	respond.WithJSON(w, http.StatusOK, response{
		User: User{
			ID:          user.ID,
			Nickname:    user.Nickname,
			RealName:    user.RealName,
			DateOfBirth: formatDateOfBirth(user.BirthDate),
			Status:      user.Status,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
		},
		Token: accessToken,
	})
}

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

func (api *API) HandlerUsersSearch(w http.ResponseWriter, r *http.Request) {
	type UserSearchResult struct {
		ID                 uuid.UUID `json:"id"`
		Nickname           string    `json:"nickname"`
		RealName           string    `json:"real_name"`
		RelationshipStatus string    `json:"relationship_status"`
	}

	type response struct {
		Users []UserSearchResult `json:"users"`
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
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
		relStatus := "none"
		rel, err := api.DB.GetRelationshipBetweenUsers(r.Context(), database.GetRelationshipBetweenUsersParams{
			ActionUserID: userID,
			TargetUserID: u.ID,
		})

		if err == nil {
			if rel.Status == "blocked" && rel.ActionUserID == u.ID {
				// Privacy rule: exclude target users who have an active block against the searching user
				continue
			}

			switch rel.Status {
			case "accepted":
				relStatus = "friends"
			case "pending":
				if rel.ActionUserID == userID {
					relStatus = "request_sent"
				} else {
					relStatus = "request_received"
				}
			case "blocked":
				if rel.ActionUserID == userID {
					relStatus = "blocked_by_me"
				}
			}
		}

		users = append(users, UserSearchResult{
			ID:                 u.ID,
			Nickname:           u.Nickname,
			RealName:           u.RealName,
			RelationshipStatus: relStatus,
		})
	}

	respond.WithJSON(w, http.StatusOK, response{
		Users: users,
	})
}

func (api *API) HandlerUserUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		respond.WithError(w, http.StatusForbidden, "Not authorized to do this", nil)
		return
	}

	type parameters struct {
		NewNickname    *string `json:"nickname,omitempty"`
		NewRealName    *string `json:"real_name,omitempty"`
		NewDateOfBirth *string `json:"date_of_birth,omitempty"`
		NewPassword    *string `json:"password,omitempty"`
		NewEmail       *string `json:"email,omitempty"`
	}

	type response struct {
		User
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if params.NewNickname == nil && params.NewRealName == nil && params.NewDateOfBirth == nil && params.NewEmail == nil && params.NewPassword == nil {
		respond.WithError(w, http.StatusBadRequest, "No fields provided to update", nil)
		return
	}

	var hashedPassword *string
	if params.NewPassword != nil && *params.NewPassword != "" {
		hash, err := auth.HashPassword(*params.NewPassword)
		if err != nil {
			respond.WithError(w, http.StatusInternalServerError, "Error hashing password", err)
			return
		}
		hashedPassword = &hash
	}

	var birthDate *time.Time
	if params.NewDateOfBirth != nil {
		parsedDob, err := parseDateOfBirth(*params.NewDateOfBirth)
		if err != nil {
			respond.WithError(w, http.StatusBadRequest, "Invalid date_of_birth format. Use YYYY-MM-DD", err)
			return
		}
		if calculateAge(parsedDob) < 18 {
			respond.WithError(w, http.StatusBadRequest, "You must be at least 18 years old to use GoChat.", nil)
			return
		}
		birthDate = &parsedDob
	}

	arg := database.UpdateUserParams{
		ID:             userID,
		Nickname:       params.NewNickname,
		RealName:       params.NewRealName,
		BirthDate:      birthDate,
		Email:          params.NewEmail,
		HashedPassword: hashedPassword,
	}

	user, err := api.DB.UpdateUser(r.Context(), arg)
	if err != nil {
		if database.IsPgErrorCode(err, "23505") {
			logrus.Warn("Conflict updating user - email or nickname exists:", err)
			respond.WithError(w, http.StatusConflict, "A user with this email or nickname already exists", err)
			return
		}
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respond.WithJSON(w, http.StatusOK, response{
		User: User{
			ID:          user.ID,
			Nickname:    user.Nickname,
			RealName:    user.RealName,
			DateOfBirth: formatDateOfBirth(user.BirthDate),
			Status:      user.Status,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
		},
	})
}

func (api *API) HandlerUserVerify(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
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

	if params.Token == "" {
		respond.WithError(w, http.StatusBadRequest, "Missing verification token", nil)
		return
	}

	incomingHash := auth.HashToken(params.Token)
	user, err := api.DB.GetUserFromVerificationToken(r.Context(), incomingHash)
	if err != nil {
		if err == sql.ErrNoRows {
			respond.WithError(w, http.StatusConflict, "Token is invalid or expired", nil)
			return
		}
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	err = api.DB.VerifyUser(r.Context(), user.ID)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	err = api.DB.DeleteVerificationTokensByUserID(r.Context(), user.ID)
	if err != nil {
		logrus.Error("Failed to delete verification token after use:", err)
	}

	type response struct {
		Message string `json:"message"`
	}
	respond.WithJSON(w, http.StatusOK, response{
		Message: "Account successfully verified. You may now log in.",
	})
}

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
		Name  string `json:"name"`
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
						RealName:       githubUser.Name,
						BirthDate:      nil,
						HashedPassword: nil,
						Status:         "active",
					})
					if err != nil {
						if database.IsPgErrorCode(err, "23505") {
							suffixedNickname := fmt.Sprintf("%s_%s", githubUser.Login, generateRandomSuffix(4))
							user, err = api.DB.CreateUser(r.Context(), database.CreateUserParams{
								ID:             uuid.Must(uuid.NewV7()),
								Email:          githubUser.Email,
								Nickname:       suffixedNickname,
								RealName:       githubUser.Name,
								BirthDate:      nil,
								HashedPassword: nil,
								Status:         "active",
							})
						}
						if err != nil {
							respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
							return
						}
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

	auth.SetAuthCookie(w, refreshToken)

	frontendURL := fmt.Sprintf("%s/oauth-callback#access_token=%s", api.FrontendURL, accessToken)

	http.Redirect(w, r, frontendURL, http.StatusFound)
}

func generateRandomSuffix(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "a7x9"
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
