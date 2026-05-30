package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kohei321dev/spot-diggz/internal/spot"
)

const sdzSpotColumns = `
spot_id,
name,
description,
lat,
lng,
tags,
visibility,
created_at,
updated_at,
deleted_at`

var _ spot.SdzStore = (*SdzStore)(nil)

type SdzStore struct {
	pool *pgxpool.Pool
}

func NewSdzStore(ctx context.Context, databaseURL string) (*SdzStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &SdzStore{pool: pool}, nil
}

func (s *SdzStore) Close() {
	s.pool.Close()
}

func (s *SdzStore) Create(ctx context.Context, input spot.SdzCreateSpotInput) (spot.SdzSpot, error) {
	id, err := spot.NewSdzID()
	if err != nil {
		return spot.SdzSpot{}, err
	}
	created, err := spot.NewSdzSpot(id, input, time.Now().UTC())
	if err != nil {
		return spot.SdzSpot{}, err
	}

	row := s.pool.QueryRow(ctx, `
INSERT INTO sdz_spots (
    spot_id,
    name,
    description,
    lat,
    lng,
    tags,
    tag_keys,
    visibility,
    created_at,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING `+sdzSpotColumns,
		created.SdzSpotID,
		created.Name,
		created.Description,
		created.SdzLocation.Lat,
		created.SdzLocation.Lng,
		created.Tags,
		tagKeys(created.Tags),
		string(created.SdzVisibility),
		created.CreatedAt,
		created.UpdatedAt,
	)
	return scanSdzSpot(row)
}

func (s *SdzStore) Get(ctx context.Context, spotID string) (spot.SdzSpot, error) {
	row := s.pool.QueryRow(ctx, `
SELECT `+sdzSpotColumns+`
FROM sdz_spots
WHERE spot_id = $1
  AND deleted_at IS NULL`,
		spotID,
	)
	return scanSdzSpot(row)
}

func (s *SdzStore) List(ctx context.Context, filter spot.SdzListFilter) ([]spot.SdzSpot, error) {
	args := make([]any, 0, 6)
	clauses := []string{"deleted_at IS NULL"}

	if filter.SdzVisibility != nil {
		args = append(args, string(*filter.SdzVisibility))
		clauses = append(clauses, fmt.Sprintf("visibility = $%d", len(args)))
	}
	if filter.SdzBBox != nil {
		bbox := filter.SdzBBox
		args = append(args, bbox.MinLng, bbox.MaxLng, bbox.MinLat, bbox.MaxLat)
		start := len(args) - 3
		clauses = append(clauses, fmt.Sprintf(
			"lng >= $%d AND lng <= $%d AND lat >= $%d AND lat <= $%d",
			start,
			start+1,
			start+2,
			start+3,
		))
	}
	if keys := tagKeys(filter.Tags); len(keys) > 0 {
		args = append(args, keys)
		clauses = append(clauses, fmt.Sprintf("tag_keys @> $%d::text[]", len(args)))
	}

	query := `
SELECT ` + sdzSpotColumns + `
FROM sdz_spots
WHERE ` + strings.Join(clauses, " AND ") + `
ORDER BY created_at DESC, spot_id ASC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list spots: %w", err)
	}
	defer rows.Close()

	spots := []spot.SdzSpot{}
	for rows.Next() {
		item, err := scanSdzSpot(rows)
		if err != nil {
			return nil, err
		}
		spots = append(spots, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spots: %w", err)
	}
	return spots, nil
}

func (s *SdzStore) Update(ctx context.Context, spotID string, input spot.SdzUpdateSpotInput) (spot.SdzSpot, error) {
	current, err := s.Get(ctx, spotID)
	if err != nil {
		return spot.SdzSpot{}, err
	}
	updated, err := spot.SdzApplyUpdate(current, input, time.Now().UTC())
	if err != nil {
		return spot.SdzSpot{}, err
	}

	row := s.pool.QueryRow(ctx, `
UPDATE sdz_spots
SET name = $1,
    description = $2,
    lat = $3,
    lng = $4,
    tags = $5,
    tag_keys = $6,
    visibility = $7,
    updated_at = $8
WHERE spot_id = $9
  AND deleted_at IS NULL
RETURNING `+sdzSpotColumns,
		updated.Name,
		updated.Description,
		updated.SdzLocation.Lat,
		updated.SdzLocation.Lng,
		updated.Tags,
		tagKeys(updated.Tags),
		string(updated.SdzVisibility),
		updated.UpdatedAt,
		spotID,
	)
	return scanSdzSpot(row)
}

func (s *SdzStore) Delete(ctx context.Context, spotID string) error {
	now := time.Now().UTC()
	commandTag, err := s.pool.Exec(ctx, `
UPDATE sdz_spots
SET deleted_at = $1,
    updated_at = $1
WHERE spot_id = $2
  AND deleted_at IS NULL`,
		now,
		spotID,
	)
	if err != nil {
		return fmt.Errorf("delete spot: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return spot.SdzErrNotFound
	}
	return nil
}

type sdzScanner interface {
	Scan(dest ...any) error
}

func scanSdzSpot(row sdzScanner) (spot.SdzSpot, error) {
	var item spot.SdzSpot
	var lat float64
	var lng float64
	var visibility string
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&item.SdzSpotID,
		&item.Name,
		&item.Description,
		&lat,
		&lng,
		&item.Tags,
		&visibility,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return spot.SdzSpot{}, spot.SdzErrNotFound
		}
		return spot.SdzSpot{}, fmt.Errorf("scan spot: %w", err)
	}

	item.SdzLocation = spot.SdzLocation{Lat: lat, Lng: lng}
	item.SdzVisibility = spot.SdzVisibility(visibility)
	if deletedAt.Valid {
		deleted := deletedAt.Time
		item.DeletedAt = &deleted
	}
	return item, nil
}

func tagKeys(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(tags))
	for _, tag := range tags {
		key := strings.ToLower(strings.TrimSpace(tag))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}
