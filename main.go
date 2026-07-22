package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"

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
	secret := os.Getenv("SECRET_TOKEN")
	if secret == "" {
		logrus.Fatal("SECRET_TOKEN must be set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		logrus.Fatal("PORT must be set")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		logrus.Fatal("Unable to connect to database: ", err)
	}
	defer conn.Close(context.Background())

	dbQueries := database.New(conn)

	api := handlers.API{
		DB:     dbQueries,
		Secret: secret,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handlers.HandlerEndpoint)
	
	mux.HandleFunc("POST /api/users", api.HandlerUserCreate)

	s := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	logrus.Info("Serving on port:", port)
	log.Fatal(s.ListenAndServe())
}
