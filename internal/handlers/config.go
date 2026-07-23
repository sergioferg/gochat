package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sergioferg/gochat/internal/database"
)

type API struct {
	DB           *database.Queries
	Pool         *pgxpool.Pool
	Secret       string
	ResendApiKey string
	BaseURL      string
}
