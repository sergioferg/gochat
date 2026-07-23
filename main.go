package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/handlers"
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

	pool := initDB(dbURL)
	defer pool.Close()

	dbQueries := database.New(pool)

	api := handlers.API{
		DB:           dbQueries,
		Secret:       secret,
		ResendApiKey: resendKey,
		BaseURL:      baseURL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handlers.HandlerEndpoint)

	mux.HandleFunc("POST /api/refresh", api.HandlerRefreshToken)
	mux.HandleFunc("POST /api/revoke", api.HandlerRevokeToken)
	mux.HandleFunc("POST /api/login", api.HandlerUserLogin)
	mux.HandleFunc("POST /api/users", api.HandlerUserCreate)
	mux.HandleFunc("POST /api/verify", api.HandlerUserVerify)

	s := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
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
