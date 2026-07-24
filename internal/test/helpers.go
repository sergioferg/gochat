package test

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDB(t *testing.T, ctx context.Context) (*pgxpool.Pool, func()) {
	dbName := "gochat"
	dbUser := "gochat_adm"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	cleanup := func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cleanup()
		t.Fatalf("failed to get connection string: %s", err)
	}

	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		cleanup()
		t.Fatalf("failed to open sql db: %s", err)
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		cleanup()
		t.Fatalf("failed to set goose dialect: %s", err)
	}

	migrationsDir := filepath.Join("..", "..", "sql", "schema")
	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		cleanup()
		t.Fatalf("failed to run goose migrations: %s", err)
	}

	dbpool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		cleanup()
		t.Fatalf("failed to create pgx pool: %s", err)
	}

	return dbpool, func() {
		dbpool.Close()
		cleanup()
	}
}
