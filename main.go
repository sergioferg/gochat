package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/justinas/alice"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/handlers"
	"github.com/sergioferg/gochat/internal/pubsub"
	"github.com/sergioferg/gochat/internal/routing"
	"github.com/sergioferg/gochat/internal/ws"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
}

func main() {
	//filePathRoot := "."

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		logrus.Fatal("DB_URL must be set")
	}
	secret := os.Getenv("JWT_SECRET_TOKEN")
	if secret == "" {
		logrus.Fatal("JWT_SECRET_TOKEN must be set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		logrus.Fatal("PORT must be set")
	}
	resendKey := os.Getenv("RESEND_API_KEY")
	if resendKey == "" {
		logrus.Fatal("RESEND_API_KEY must be set")
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		logrus.Fatal("PLATFORM must be set")
	}
	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		logrus.Fatal("RABBITMQ_URL must be set")
	}

	var githubOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  fmt.Sprintf("%s/api/oauth/github/callback", baseURL),
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}

	pool := initDB(dbURL)
	defer pool.Close()

	dbQueries := database.New(pool)
	wsManager := ws.NewManager()

	rabbitClient, err := pubsub.New(rmqURL)
	if err != nil {
		logrus.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer rabbitClient.Close()

	api := handlers.API{
		DB:             dbQueries,
		Pool:           pool,
		WSManager:      wsManager,
		RMQ:            rabbitClient,
		GithubOauthCfg: githubOAuthConfig,
		Secret:         secret,
		ResendApiKey:   resendKey,
		BaseURL:        baseURL,
	}

	// Start listening in the background
	err = pubsub.SubscribeJSON(
		rabbitClient.Conn,
		routing.ChatPrefix,
		"", // Random temporary queue
		"", // Empty routing key
		pubsub.Transient,
		func(event routing.ChatEvent) pubsub.AckType {
			if event.Type == "new_message" {
				for _, userID := range event.TargetUserIDs {
					api.WSManager.SendToUser(userID, event)
				}
			}
			return pubsub.Ack
		},
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handlers.HandlerEndpoint)

	// Refresh JWT
	mux.HandleFunc("POST /api/refresh", api.HandlerRefreshAccessToken)

	// User auth
	mux.HandleFunc("POST /api/login", api.HandlerUserLogin)
	mux.HandleFunc("POST /api/users", api.HandlerUserCreate)
	mux.HandleFunc("POST /api/verify", api.HandlerUserVerify)

	// Session termination
	mux.HandleFunc("POST /api/logout", api.HandlerUserLogout)

	// Github auth/session
	mux.HandleFunc("GET /api/oauth/github/login", api.HandlerGitHubLogin)
	mux.HandleFunc("GET /api/oauth/github/callback", api.HandlerGitHubCallback)

	protectedChain := alice.New(api.AuthMiddleware)

	// Active session management
	mux.Handle("GET /api/sessions", protectedChain.ThenFunc(api.HandlerGetSessions))
	mux.Handle("DELETE /api/sessions/{id}", protectedChain.ThenFunc(api.HandlerRevokeSession))

	// Delete and Update users
	mux.Handle("DELETE /api/users", protectedChain.ThenFunc(api.HandlerUserDelete))
	mux.Handle("PATCH /api/users", protectedChain.ThenFunc(api.HandlerUserUpdate))

	// Real-time connections
	mux.Handle("GET /api/ws", protectedChain.ThenFunc(api.HandlerWebSocket))
	mux.Handle("POST /api/messages", protectedChain.ThenFunc(api.HandlerSendMessage))

	globalChain := alice.New(api.SecurityHeadersMiddleware)

	s := &http.Server{
		Addr:         ":" + port,
		Handler:      globalChain.Then(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logrus.Info("Serving on port:", port)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatal("Server failed:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		logrus.Fatal("Server forced to shutdown:", err)
	}

	logrus.Info("Server exited properly")
}

func initDB(connString string) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		logrus.Fatal("Failed to parse config:", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute
	config.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		logrus.Fatal("Failed to create pool:", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		logrus.Fatal("Failed to ping database:", err)
	}

	return pool
}
