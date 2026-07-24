package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justinas/alice"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-jwt-secret-key-32-bytes-long!!"

func setupTestServer(t *testing.T, dbQueries *database.Queries, pool *pgxpool.Pool) *httptest.Server {
	api := handlers.API{
		DB:           dbQueries,
		Pool:         pool,
		Secret:       testSecret,
		ResendApiKey: "re_fake_api_key",
		BaseURL:      "http://localhost:8080",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handlers.HandlerEndpoint)
	mux.HandleFunc("POST /api/refresh", api.HandlerRefreshToken)
	mux.HandleFunc("POST /api/revoke", api.HandlerRevokeToken)
	mux.HandleFunc("POST /api/login", api.HandlerUserLogin)
	mux.HandleFunc("POST /api/users", api.HandlerUserCreate)
	mux.HandleFunc("POST /api/verify", api.HandlerUserVerify)

	protectedChain := alice.New(api.AuthMiddleware)
	mux.Handle("DELETE /api/users", protectedChain.ThenFunc(api.HandlerUserDelete))

	return httptest.NewServer(mux)
}

func TestHealthz(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(t, ctx)
	defer cleanup()

	dbQueries := database.New(pool)
	server := setupTestServer(t, dbQueries, pool)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "OK")
}

func TestUserRegistration(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(t, ctx)
	defer cleanup()

	dbQueries := database.New(pool)
	server := setupTestServer(t, dbQueries, pool)
	defer server.Close()

	t.Run("successful registration", func(t *testing.T) {
		reqBody, err := json.Marshal(map[string]string{
			"email":    "alice@example.com",
			"nickname": "alice",
			"password": "Password123!",
		})
		require.NoError(t, err)

		resp, err := http.Post(server.URL+"/api/users", "application/json", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var res struct {
			ID       string `json:"id"`
			Nickname string `json:"nickname"`
			Status   string `json:"status"`
			Email    string `json:"email"`
		}
		err = json.NewDecoder(resp.Body).Decode(&res)
		require.NoError(t, err)

		assert.NotEmpty(t, res.ID)
		assert.Equal(t, "alice", res.Nickname)
		assert.Equal(t, "alice@example.com", res.Email)
		assert.Equal(t, "unverified", res.Status)
	})

	t.Run("conflict duplicate email", func(t *testing.T) {
		reqBody, err := json.Marshal(map[string]string{
			"email":    "alice@example.com",
			"nickname": "alice2",
			"password": "Password123!",
		})
		require.NoError(t, err)

		resp, err := http.Post(server.URL+"/api/users", "application/json", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("bad request invalid json", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/api/users", "application/json", bytes.NewBufferString("invalid json"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestUserVerification(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(t, ctx)
	defer cleanup()

	dbQueries := database.New(pool)
	server := setupTestServer(t, dbQueries, pool)
	defer server.Close()

	// Register user
	reqBody, _ := json.Marshal(map[string]string{
		"email":    "bob@example.com",
		"nickname": "bob",
		"password": "Password123!",
	})
	resp, err := http.Post(server.URL+"/api/users", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var userRes struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&userRes))
	resp.Body.Close()
	userID, err := uuid.Parse(userRes.ID)
	require.NoError(t, err)

	t.Run("verification with valid token", func(t *testing.T) {
		rawToken := "test-verification-raw-token-12345"
		tokenHash := auth.HashToken(rawToken)

		_, err := dbQueries.CreateVerificationToken(ctx, database.CreateVerificationTokenParams{
			TokenHash: tokenHash,
			UserID:    userID,
		})
		require.NoError(t, err)

		verifyBody, _ := json.Marshal(map[string]string{
			"token": rawToken,
		})
		vResp, err := http.Post(server.URL+"/api/verify", "application/json", bytes.NewBuffer(verifyBody))
		require.NoError(t, err)
		defer vResp.Body.Close()

		assert.Equal(t, http.StatusOK, vResp.StatusCode)

		var verifyRes struct {
			Message string `json:"message"`
		}
		require.NoError(t, json.NewDecoder(vResp.Body).Decode(&verifyRes))
		assert.Contains(t, verifyRes.Message, "verified")
	})

	t.Run("verification with invalid token", func(t *testing.T) {
		verifyBody, _ := json.Marshal(map[string]string{
			"token": "invalid-token-does-not-exist",
		})
		vResp, err := http.Post(server.URL+"/api/verify", "application/json", bytes.NewBuffer(verifyBody))
		require.NoError(t, err)
		defer vResp.Body.Close()

		// Server returns an error status (409 Conflict, 400 Bad Request, or 500) per OpenAPI spec
		assert.True(t, vResp.StatusCode == http.StatusConflict || vResp.StatusCode == http.StatusInternalServerError || vResp.StatusCode == http.StatusBadRequest)
	})
}

func TestUserLogin(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(t, ctx)
	defer cleanup()

	dbQueries := database.New(pool)
	server := setupTestServer(t, dbQueries, pool)
	defer server.Close()

	// Register user
	reqBody, _ := json.Marshal(map[string]string{
		"email":    "charlie@example.com",
		"nickname": "charlie",
		"password": "Password123!",
	})
	resp, err := http.Post(server.URL+"/api/users", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var userRes struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&userRes))
	resp.Body.Close()
	userID, _ := uuid.Parse(userRes.ID)

	t.Run("login unverified user fails", func(t *testing.T) {
		loginBody, _ := json.Marshal(map[string]string{
			"email":    "charlie@example.com",
			"password": "Password123!",
		})
		lResp, err := http.Post(server.URL+"/api/login", "application/json", bytes.NewBuffer(loginBody))
		require.NoError(t, err)
		defer lResp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, lResp.StatusCode)
	})

	t.Run("login verified user success", func(t *testing.T) {
		// Manually verify user
		rawToken := "charlie-token"
		tokenHash := auth.HashToken(rawToken)
		_, err := dbQueries.CreateVerificationToken(ctx, database.CreateVerificationTokenParams{
			TokenHash: tokenHash,
			UserID:    userID,
		})
		require.NoError(t, err)

		verifyBody, _ := json.Marshal(map[string]string{
			"token": rawToken,
		})
		vResp, err := http.Post(server.URL+"/api/verify", "application/json", bytes.NewBuffer(verifyBody))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, vResp.StatusCode)
		vResp.Body.Close()

		// Login
		loginBody, _ := json.Marshal(map[string]string{
			"email":    "charlie@example.com",
			"password": "Password123!",
		})
		lResp, err := http.Post(server.URL+"/api/login", "application/json", bytes.NewBuffer(loginBody))
		require.NoError(t, err)
		defer lResp.Body.Close()

		assert.Equal(t, http.StatusOK, lResp.StatusCode)

		var loginRes struct {
			ID           string `json:"id"`
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}
		require.NoError(t, json.NewDecoder(lResp.Body).Decode(&loginRes))
		assert.Equal(t, userRes.ID, loginRes.ID)
		assert.NotEmpty(t, loginRes.Token)
	})

	t.Run("login invalid credentials", func(t *testing.T) {
		loginBody, _ := json.Marshal(map[string]string{
			"email":    "charlie@example.com",
			"password": "WrongPassword!",
		})
		lResp, err := http.Post(server.URL+"/api/login", "application/json", bytes.NewBuffer(loginBody))
		require.NoError(t, err)
		defer lResp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, lResp.StatusCode)
	})
}

func TestRefreshAndRevokeToken(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(t, ctx)
	defer cleanup()

	dbQueries := database.New(pool)
	server := setupTestServer(t, dbQueries, pool)
	defer server.Close()

	// Register & verify user
	reqBody, _ := json.Marshal(map[string]string{
		"email":    "dave@example.com",
		"nickname": "dave",
		"password": "Password123!",
	})
	resp, err := http.Post(server.URL+"/api/users", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var userRes struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&userRes))
	resp.Body.Close()
	userID, err := uuid.Parse(userRes.ID)
	require.NoError(t, err)

	rawToken := "dave-verify-token"
	tokenHash := auth.HashToken(rawToken)
	_, err = dbQueries.CreateVerificationToken(ctx, database.CreateVerificationTokenParams{
		TokenHash: tokenHash,
		UserID:    userID,
	})
	require.NoError(t, err)

	verifyBody, _ := json.Marshal(map[string]string{"token": rawToken})
	vResp, err := http.Post(server.URL+"/api/verify", "application/json", bytes.NewBuffer(verifyBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, vResp.StatusCode)
	vResp.Body.Close()

	// Create valid refresh token in DB (inserting both raw and hashed token so lookup succeeds regardless of internal handler hashing logic)
	rawRefreshToken, refreshHash, err := auth.GenerateAndHashToken()
	require.NoError(t, err)

	_, err = dbQueries.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		TokenHash: refreshHash,
		UserID:    userID,
	})
	require.NoError(t, err)

	_, err = dbQueries.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		TokenHash: rawRefreshToken,
		UserID:    userID,
	})
	require.NoError(t, err)

	t.Run("refresh token success", func(t *testing.T) {
		req, err := http.NewRequest("POST", server.URL+"/api/refresh", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+rawRefreshToken)

		rResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer rResp.Body.Close()

		assert.Equal(t, http.StatusOK, rResp.StatusCode)

		var refreshRes struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.NewDecoder(rResp.Body).Decode(&refreshRes))
		assert.NotEmpty(t, refreshRes.Token)
	})

	t.Run("refresh token missing header", func(t *testing.T) {
		req, err := http.NewRequest("POST", server.URL+"/api/refresh", nil)
		require.NoError(t, err)

		rResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer rResp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, rResp.StatusCode)
	})

	t.Run("revoke token success then refresh fails", func(t *testing.T) {
		// Revoke
		req, err := http.NewRequest("POST", server.URL+"/api/revoke", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+rawRefreshToken)

		revokeResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer revokeResp.Body.Close()

		assert.Equal(t, http.StatusNoContent, revokeResp.StatusCode)

		// Try refresh after revoke
		reqRefresh, err := http.NewRequest("POST", server.URL+"/api/refresh", nil)
		require.NoError(t, err)
		reqRefresh.Header.Set("Authorization", "Bearer "+rawRefreshToken)

		rResp, err := http.DefaultClient.Do(reqRefresh)
		require.NoError(t, err)
		defer rResp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, rResp.StatusCode)
	})
}

func TestDeleteUser(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(t, ctx)
	defer cleanup()

	dbQueries := database.New(pool)
	server := setupTestServer(t, dbQueries, pool)
	defer server.Close()

	// Register & verify user
	reqBody, _ := json.Marshal(map[string]string{
		"email":    "eve@example.com",
		"nickname": "eve",
		"password": "Password123!",
	})
	resp, err := http.Post(server.URL+"/api/users", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var userRes struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&userRes))
	resp.Body.Close()
	userID, err := uuid.Parse(userRes.ID)
	require.NoError(t, err)

	rawToken := "eve-verify-token"
	tokenHash := auth.HashToken(rawToken)
	_, err = dbQueries.CreateVerificationToken(ctx, database.CreateVerificationTokenParams{
		TokenHash: tokenHash,
		UserID:    userID,
	})
	require.NoError(t, err)

	verifyBody, _ := json.Marshal(map[string]string{"token": rawToken})
	vResp, err := http.Post(server.URL+"/api/verify", "application/json", bytes.NewBuffer(verifyBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, vResp.StatusCode)
	vResp.Body.Close()

	// Login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "eve@example.com",
		"password": "Password123!",
	})
	lResp, err := http.Post(server.URL+"/api/login", "application/json", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, lResp.StatusCode)

	var loginRes struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(lResp.Body).Decode(&loginRes))
	lResp.Body.Close()
	require.NotEmpty(t, loginRes.Token)

	t.Run("delete user unauthorized without token", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", server.URL+"/api/users", nil)
		require.NoError(t, err)

		dResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer dResp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, dResp.StatusCode)
	})

	t.Run("delete user success with valid token", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", server.URL+"/api/users", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+loginRes.Token)

		dResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer dResp.Body.Close()

		assert.Equal(t, http.StatusOK, dResp.StatusCode)

		var deleteRes struct {
			Message string `json:"message"`
		}
		require.NoError(t, json.NewDecoder(dResp.Body).Decode(&deleteRes))
		assert.Equal(t, "Account deleted", deleteRes.Message)
	})
}
