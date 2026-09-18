package api

import (
	"log"
	"net"
	"net/http"
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
			log.Printf("rate limiter error, failing open: %v", err)
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
