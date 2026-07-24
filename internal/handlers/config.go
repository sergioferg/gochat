package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sergioferg/gochat/internal/database"
	"golang.org/x/oauth2"
)

type API struct {
	DB             *database.Queries
	Pool           *pgxpool.Pool
	GithubOauthCfg *oauth2.Config
	Secret         string
	ResendApiKey   string
	BaseURL        string
	Platform       string
}
