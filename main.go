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
	"github.com/sergioferg/gochat/internal/middleware"
	"github.com/sergioferg/gochat/internal/pubsub"
	"github.com/sergioferg/gochat/internal/routing"
	"github.com/sergioferg/gochat/internal/routes"
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
	backURL := os.Getenv("BACK_URL")
	if backURL == "" {
		backURL = "http://localhost:8080"
	}
	frontURL := os.Getenv("FRONT_URL")
	if frontURL == "" {
		frontURL = "http://localhost:5173"
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
		RedirectURL:  fmt.Sprintf("%s/oauth/github/callback", backURL),
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
		BackendURL:     backURL,
		FrontendURL:    frontURL,
		Platform:       platform,
	}

	mw := middleware.Config{
		DB:       dbQueries,
		Secret:   secret,
		Platform: platform,
	}

	// Start listening in the background
	err = pubsub.SubscribeJSON(
		rabbitClient.Conn,
		routing.ChatPrefix,
		"", // Random temporary queue
		"", // Empty routing key
		pubsub.Transient,
		func(event routing.ChatEvent) pubsub.AckType {
			for _, userID := range event.TargetUserIDs {
				api.WSManager.SendToUser(userID, event)
			}
			return pubsub.Ack
		},
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.HandlerEndpoint)

	authChain := alice.New(mw.AuthMiddleware)
	fullAccountChain := alice.New(mw.AuthMiddleware, mw.RequireCompletedProfile)

	routes.SetupAuthRoutes(mux, &api, authChain, fullAccountChain)
	routes.SetupUserRoutes(mux, &api, authChain, fullAccountChain)
	routes.SetupChatRoutes(mux, &api, authChain, fullAccountChain)
	routes.SetupRequestRoutes(mux, &api, authChain, fullAccountChain)
	routes.SetupWebSocketRoutes(mux, &api, authChain, fullAccountChain)

	globalChain := alice.New(middleware.CORSMiddleware, mw.SecurityHeadersMiddleware)

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
