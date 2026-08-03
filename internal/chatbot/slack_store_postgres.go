package chatbot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	slackStoreQueryTimeout = 3 * time.Second
)

type PostgresSlackRequestStore struct {
	db *sql.DB
}

func NewPostgresSlackRequestStore(databaseURL string) (*PostgresSlackRequestStore, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("Slack request store requires a database URL")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open Slack request store: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), slackStoreQueryTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping Slack request store: %w", err)
	}
	return &PostgresSlackRequestStore{db: db}, nil
}

func (store *PostgresSlackRequestStore) Begin(ctx context.Context, request SlackRecommendationRequest) (bool, error) {
	result, err := store.db.ExecContext(ctx, `
		INSERT INTO slack_recommendation_requests (
			request_id, source_event_key, status, created_at, updated_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (source_event_key) DO NOTHING
	`, request.RequestID, request.SourceEventKey, SlackRequestGenerating,
		request.CreatedAt.UTC(), request.UpdatedAt.UTC(), request.ExpiresAt.UTC())
	if err != nil {
		return false, fmt.Errorf("insert Slack recommendation request: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read Slack recommendation insert result: %w", err)
	}
	return rows == 1, nil
}

func (store *PostgresSlackRequestStore) MarkDelivered(ctx context.Context, requestID string, at time.Time) error {
	result, err := store.db.ExecContext(ctx, `
		UPDATE slack_recommendation_requests
		SET status = $2, updated_at = $3
		WHERE request_id = $1 AND status = $4
	`, requestID, SlackRequestDelivered, at.UTC(), SlackRequestGenerating)
	return requireOneSlackRequestRow(result, err, "mark Slack request delivered")
}

func (store *PostgresSlackRequestStore) MarkFailed(ctx context.Context, requestID string, at time.Time) error {
	result, err := store.db.ExecContext(ctx, `
		UPDATE slack_recommendation_requests SET status = $2, updated_at = $3
		WHERE request_id = $1 AND status = $4
	`, requestID, SlackRequestFailed, at.UTC(), SlackRequestGenerating)
	return requireOneSlackRequestRow(result, err, "mark Slack request failed")
}

func (store *PostgresSlackRequestStore) PurgeExpired(ctx context.Context, at time.Time) error {
	_, err := store.db.ExecContext(ctx, `DELETE FROM slack_recommendation_requests WHERE expires_at <= $1`, at.UTC())
	if err != nil {
		return fmt.Errorf("purge expired Slack recommendation requests: %w", err)
	}
	return nil
}

func (store *PostgresSlackRequestStore) Close() error {
	if store == nil || store.db == nil {
		return nil
	}
	return store.db.Close()
}

func requireOneSlackRequestRow(result sql.Result, err error, operation string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s result: %w", operation, err)
	}
	if rows != 1 {
		return ErrSlackRequestConflict
	}
	return nil
}
