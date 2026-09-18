// Package logging provides a single structured (JSON) logger for the
// service, plus a way to carry a request ID through a context so a
// redirect's log line and the async click-write it triggered can later
// be correlated by grepping for the same request_id.
package logging

import (
	"context"
	"log/slog"
	"os"
)

var base = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type ctxKey struct{}

// WithRequestID returns a context carrying a logger pre-tagged with the
// given request ID, so every log line taken from this context onward
// includes it automatically.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, base.With("request_id", requestID))
}

// From returns the request-scoped logger, falling back to the base
// logger for background work (e.g. the click batcher) that has no
// single request to attribute a log line to.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return base
}

// Base returns the unscoped logger, for use outside any request context.
func Base() *slog.Logger {
	return base
}
