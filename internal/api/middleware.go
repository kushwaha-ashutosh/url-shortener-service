package api

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"time"

	"github.com/kushwaha-ashutosh/url-shortener/internal/logging"
)

// RateLimit gates a route behind the handler's Redis-backed limiter,
// keyed by client IP. On a limiter error (e.g. Redis is briefly
// unreachable) it fails open and logs the failure: a degraded rate
// limiter shouldn't take down link creation entirely, since the limiter
// exists to shed abusive load, not to be a second source of truth for
// whether the service can accept writes.
func (h *Handler) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)

		allowed, err := h.limiter.Allow(r.Context(), key)
		if err != nil {
			logging.From(r.Context()).Warn("rate limiter error, failing open", "error", err)
			next.ServeHTTP(w, r)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded, try again shortly")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// CORS allows the dashboard (served from a different origin during
// local dev, e.g. Vite on :5173) to call this API from the browser.
// The API has no auth and no cookies to protect, so a single
// configurable allowed origin is enough — this isn't guarding
// anything sensitive, just satisfying the browser's same-origin
// policy for a fetch() call.
func (h *Handler) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", h.allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestID tags every request with an ID — reusing one supplied via
// X-Request-Id (so a request can be traced across services behind a
// gateway) or generating one otherwise — echoes it back in the
// response, and logs one structured line per request. Every log line
// produced further down the chain via logging.From(r.Context()) is
// automatically tagged with the same ID, so a redirect's log entry and
// the async click-write it triggered can be correlated later.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = generateRequestID()
		}
		w.Header().Set("X-Request-Id", id)

		ctx := logging.WithRequestID(r.Context(), id)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		start := time.Now()
		next.ServeHTTP(rec, r.WithContext(ctx))

		logging.From(ctx).Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
