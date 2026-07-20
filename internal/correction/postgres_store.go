package correction

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	postgresPingTimeout  = 5 * time.Second
	postgresQueryTimeout = 5 * time.Second
)

// PostgresStore persists correction reports in Neon or another PostgreSQL-compatible service.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(databaseURL string, now time.Time) (*PostgresStore, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("%w: database URL is required", ErrStoreUnavailable)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres correction store: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	store := &PostgresStore{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), postgresPingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres correction store: %w", err)
	}
	if err := store.PurgeExpired(now.UTC()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (store *PostgresStore) Save(ctx context.Context, report Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := store.db.ExecContext(ctx, `
		INSERT INTO correction_reports (
			report_id,
			facility_id,
			category,
			details,
			evidence_url,
			contact,
			contact_consent,
			received_at,
			delete_after
		) VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9)
	`, report.ReportID, report.FacilityID, report.Category, report.Details,
		report.EvidenceURL, report.Contact, report.ContactConsent,
		report.ReceivedAt.UTC(), report.DeleteAfter.UTC())
	if err != nil {
		return fmt.Errorf("insert correction report: %w", err)
	}
	return nil
}

func (store *PostgresStore) PurgeExpired(now time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), postgresQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx,
		`DELETE FROM correction_reports WHERE delete_after <= $1`, now.UTC()); err != nil {
		return fmt.Errorf("purge expired correction reports: %w", err)
	}
	return nil
}

func (store *PostgresStore) Close() error {
	if store == nil || store.db == nil {
		return nil
	}
	return store.db.Close()
}
