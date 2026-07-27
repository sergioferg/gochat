package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/pubsub"
	"github.com/sergioferg/gochat/internal/ws"
	"golang.org/x/oauth2"
)

type API struct {
	DB             *database.Queries
	Pool           *pgxpool.Pool
	WSManager      *ws.Manager
	RMQ            *pubsub.RabbitMQ
	GithubOauthCfg *oauth2.Config
	Secret         string
	ResendApiKey   string
	BackendURL     string
	FrontendURL    string
	Platform       string
}
