package middleware

import "github.com/sergioferg/gochat/internal/database"

type Config struct {
	DB       *database.Queries
	Secret   string
	Platform string
}
