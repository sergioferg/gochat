package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func IsPgErrorCode(err error, code string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == code
}
