package handlers

import "github.com/sergioferg/gochat/internal/database"

type API struct {
	DB     *database.Queries
	Secret string
}
