// Package repository handles all database interactions.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors returned by repository methods.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("short code already exists")
)

// URL represents a stored URL mapping.
type URL struct {
	ID        int64     `db:"id"`
	ShortCode string    `db:"short_code"`
	LongURL   string    `db:"long_url"`
	CreatedAt time.Time `db:"created_at"`
}

// Repository handles database operations for URL records.
type Repository struct {
	db *pgxpool.Pool
}

// New creates a Repository with the given connection pool.
func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateSchema creates the urls table if it does not already exist.
// In production this would be replaced with a migration tool like goose or atlas.
func (r *Repository) CreateSchema(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS urls (
			id         BIGSERIAL    PRIMARY KEY,
			short_code VARCHAR(10)  NOT NULL UNIQUE,
			long_url   TEXT         NOT NULL,
			created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	return nil
}

// Save inserts a new URL mapping and returns the created record.
func (r *Repository) Save(ctx context.Context, shortCode, longURL string) (URL, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO urls (short_code, long_url)
		VALUES ($1, $2)
		RETURNING id, short_code, long_url, created_at
	`, shortCode, longURL)
	if err != nil {
		return URL{}, translateError(err, "inserting url")
	}

	url, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[URL])
	if err != nil {
		return URL{}, translateError(err, "scanning inserted url")
	}
	return url, nil
}

// FindByCode retrieves a URL by its short code. Returns ErrNotFound if missing.
func (r *Repository) FindByCode(ctx context.Context, shortCode string) (URL, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, short_code, long_url, created_at
		FROM urls
		WHERE short_code = $1
	`, shortCode)
	if err != nil {
		return URL{}, translateError(err, "querying url")
	}

	url, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[URL])
	if err != nil {
		return URL{}, translateError(err, "scanning url")
	}
	return url, nil
}

// translateError maps pgx errors to repository sentinels.
func translateError(err error, op string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}

	return fmt.Errorf("%s: %w", op, err)
}
