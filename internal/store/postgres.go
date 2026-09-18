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
	Code         string
	TotalClicks  int64
	ClicksByDay  []DailyClicks
	TopReferrers []NamedCount
	TopBrowsers  []NamedCount
}

type DailyClicks struct {
	Day    string
	Clicks int64
}

type NamedCount struct {
	Name   string
	Clicks int64
}

const statsWindowDays = 30

// GetStats aggregates click history for a code: a running total, a
// per-day series over the last 30 days (for a trend line), a
// referrer breakdown (grouping empty referrers as "direct" traffic),
// and a browser-family breakdown. The browser classification runs as
// SQL rather than pulling every raw user-agent string back to Go,
// since the aggregation is what's needed, not the raw rows.
func (s *Store) GetStats(ctx context.Context, code string) (*Stats, error) {
	stats := &Stats{Code: code}

	const totalQ = `SELECT count(*) FROM clicks WHERE code = $1`
	if err := s.pool.QueryRow(ctx, totalQ, code).Scan(&stats.TotalClicks); err != nil {
		return nil, err
	}

	const byDayQ = `
		SELECT date_trunc('day', clicked_at)::date AS day, count(*)
		FROM clicks
		WHERE code = $1 AND clicked_at >= now() - make_interval(days => $2)
		GROUP BY day
		ORDER BY day
	`
	byDayRows, err := s.pool.Query(ctx, byDayQ, code, statsWindowDays)
	if err != nil {
		return nil, err
	}
	defer byDayRows.Close()
	for byDayRows.Next() {
		var day time.Time
		var count int64
		if err := byDayRows.Scan(&day, &count); err != nil {
			return nil, err
		}
		stats.ClicksByDay = append(stats.ClicksByDay, DailyClicks{Day: day.Format("2006-01-02"), Clicks: count})
	}
	if err := byDayRows.Err(); err != nil {
		return nil, err
	}

	const referrersQ = `
		SELECT COALESCE(NULLIF(referrer, ''), 'direct') AS referrer, count(*)
		FROM clicks
		WHERE code = $1
		GROUP BY referrer
		ORDER BY count(*) DESC
		LIMIT 10
	`
	stats.TopReferrers, err = s.namedCounts(ctx, referrersQ, code)
	if err != nil {
		return nil, err
	}

	const browsersQ = `
		SELECT
			CASE
				WHEN user_agent ILIKE '%edg/%' THEN 'Edge'
				WHEN user_agent ILIKE '%chrome%' AND user_agent NOT ILIKE '%edg/%' THEN 'Chrome'
				WHEN user_agent ILIKE '%firefox%' THEN 'Firefox'
				WHEN user_agent ILIKE '%safari%' AND user_agent NOT ILIKE '%chrome%' THEN 'Safari'
				WHEN user_agent = '' OR user_agent ILIKE '%curl%' OR user_agent ILIKE '%bot%' THEN 'Bot/CLI'
				ELSE 'Other'
			END AS browser,
			count(*)
		FROM clicks
		WHERE code = $1
		GROUP BY browser
		ORDER BY count(*) DESC
	`
	stats.TopBrowsers, err = s.namedCounts(ctx, browsersQ, code)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *Store) namedCounts(ctx context.Context, query, code string) ([]NamedCount, error) {
	rows, err := s.pool.Query(ctx, query, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []NamedCount
	for rows.Next() {
		var c NamedCount
		if err := rows.Scan(&c.Name, &c.Clicks); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
