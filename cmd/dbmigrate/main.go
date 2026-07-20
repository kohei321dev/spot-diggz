package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	databaseURLEnv = "DATABASE_URL"
	migrationPath  = "db/migrations/0001-correction-reports.sql"
	queryTimeout   = 10 * time.Second
)

func main() {
	databaseURL := strings.TrimSpace(os.Getenv(databaseURLEnv))
	if databaseURL == "" {
		fail(errors.New("DATABASE_URL is required"))
	}

	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		fail(fmt.Errorf("read migration: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		fail(fmt.Errorf("connect database: %w", err))
	}
	defer connection.Close(context.Background())

	if _, err := connection.Exec(ctx, string(migration)); err != nil {
		fail(fmt.Errorf("apply migration: %w", err))
	}
	fmt.Printf("Applied %s\n", migrationPath)
}

func fail(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "dbmigrate: %v\n", err)
	os.Exit(1)
}
