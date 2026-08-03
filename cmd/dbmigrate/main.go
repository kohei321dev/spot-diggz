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
	queryTimeout   = 10 * time.Second
)

var migrationPaths = []string{
	"db/migrations/0001-correction-reports.sql",
	"db/migrations/0002-slack-recommendation-requests.sql",
	"db/migrations/0003-remove-saved-facilities.sql",
}

func main() {
	databaseURL := strings.TrimSpace(os.Getenv(databaseURLEnv))
	if databaseURL == "" {
		fail(errors.New("DATABASE_URL is required"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		fail(fmt.Errorf("connect database: %w", err))
	}
	defer connection.Close(context.Background())

	for _, migrationPath := range migrationPaths {
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			fail(fmt.Errorf("read migration %s: %w", migrationPath, err))
		}
		if _, err := connection.Exec(ctx, string(migration)); err != nil {
			fail(fmt.Errorf("apply migration %s: %w", migrationPath, err))
		}
		fmt.Printf("Applied %s\n", migrationPath)
	}
}

func fail(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "dbmigrate: %v\n", err)
	os.Exit(1)
}
