// Package store handles persistence for links and click events.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a link code has no matching row.
var ErrNotFound = errors.New("link not found")

// ErrCodeTaken is returned when a custom code already exists.
var ErrCodeTaken = errors.New("code already in use")

type Link struct {
	Code      string
	LongURL   string
	CreatedAt time.Time
}

type ClickEvent struct {
	Code      string
	Timestamp time.Time
	Referrer  string
	UserAgent string
}

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

// CreateLink inserts a new link. Returns ErrCodeTaken on a unique
// constraint violation so callers can retry with a fresh code.
func (s *Store) CreateLink(ctx context.Context, code, longURL string) (*Link, error) {
	const q = `
		INSERT INTO links (code, long_url, created_at)
		VALUES ($1, $2, now())
		RETURNING code, long_url, created_at
	`
	var l Link
	err := s.pool.QueryRow(ctx, q, code, longURL).Scan(&l.Code, &l.LongURL, &l.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCodeTaken
		}
		return nil, err
	}
	return &l, nil
}

func (s *Store) GetLink(ctx context.Context, code string) (*Link, error) {
	const q = `SELECT code, long_url, created_at FROM links WHERE code = $1`
	var l Link
	err := s.pool.QueryRow(ctx, q, code).Scan(&l.Code, &l.LongURL, &l.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

// InsertClicks batch-inserts click events using pgx's CopyFrom, which is
// far cheaper than one INSERT per row when flushing a buffered batch.
func (s *Store) InsertClicks(ctx context.Context, events []ClickEvent) error {
	if len(events) == 0 {
		return nil
	}
	rows := make([][]any, len(events))
	for i, e := range events {
		rows[i] = []any{e.Code, e.Timestamp, e.Referrer, e.UserAgent}
	}
	_, err := s.pool.CopyFrom(
		ctx,
		pgx.Identifier{"clicks"},
		[]string{"code", "clicked_at", "referrer", "user_agent"},
		pgx.CopyFromRows(rows),
	)
	return err
}

type Stats struct {
	Code        string
	TotalClicks int64
}

func (s *Store) GetStats(ctx context.Context, code string) (*Stats, error) {
	const q = `SELECT count(*) FROM clicks WHERE code = $1`
	var count int64
	if err := s.pool.QueryRow(ctx, q, code).Scan(&count); err != nil {
		return nil, err
	}
	return &Stats{Code: code, TotalClicks: count}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
