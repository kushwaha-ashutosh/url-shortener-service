// Package cache implements the cache-aside layer in front of Postgres
// for hot redirect lookups.
package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrMiss = errors.New("cache miss")

const linkTTL = 24 * time.Hour

// No retries, and client-level timeouts loose enough for a real
// cross-region TLS connection (Render talking to Upstash over the
// public internet), but still bounded rather than left at go-redis's
// defaults (5s dial / 3s read+write, 3 retries with backoff), which
// let a single request hang for 20-30+ seconds when Redis is down
// before the cache-aside fallback ever gets a chance to run — found by
// killing Redis under load in local Docker Compose testing.
//
// That local test tuned these down to 250ms, which then broke the
// very first production deploy: Render-to-Upstash's TLS handshake
// over real internet latency took longer than 250ms and got cut off
// mid-handshake, surfacing as a bare "EOF" rather than a timeout.
// Sub-millisecond LAN latency and real cross-region latency are not
// the same thing, and tuning against only the former missed it.
//
// The tighter, request-scoped bound that actually matters for the
// chaos-tested "Redis is down" scenario lives at the call site
// instead — see cacheCallTimeout in internal/api/handlers.go, which
// wraps each Redis call on the redirect hot path in its own shorter
// context deadline. These client-level timeouts just need to be loose
// enough not to break a legitimately slow-but-working connection.
const (
	dialTimeout  = 2 * time.Second
	readTimeout  = 1 * time.Second
	writeTimeout = 1 * time.Second
)

type Cache struct {
	rdb *redis.Client
}

// New accepts either a plain "host:port" (local dev, e.g. Docker
// Compose's redis service) or a full connection URL — "redis://" or
// "rediss://" for a TLS-requiring managed provider like Upstash,
// which is how REDIS_ADDR needs to look in production.
func New(addrOrURL string) (*Cache, error) {
	var opt *redis.Options
	if strings.Contains(addrOrURL, "://") {
		parsed, err := redis.ParseURL(addrOrURL)
		if err != nil {
			return nil, fmt.Errorf("invalid redis URL: %w", err)
		}
		opt = parsed
	} else {
		opt = &redis.Options{Addr: addrOrURL}
	}
	opt.DialTimeout = dialTimeout
	opt.ReadTimeout = readTimeout
	opt.WriteTimeout = writeTimeout
	opt.MaxRetries = -1 // disabled: a stuck retry loop defeats the point of failing fast

	// go-redis v9 defaults to Protocol 3 (RESP3), sending a HELLO 3
	// command right after connecting to negotiate it. Found the hard
	// way in production: TCP and TLS both completed cleanly against
	// Upstash (confirmed independently, bypassing this client entirely
	// — see cmd/server/main.go's raw TCP/TLS diagnostics), but every
	// connection still failed with a bare EOF immediately after the
	// TLS handshake. That's the signature of a managed/proxied Redis
	// backend that only understands the older RESP2 protocol and
	// simply closes the connection on an unrecognized HELLO, rather
	// than replying with a proper protocol error. Forcing RESP2 (the
	// traditional AUTH-command flow, universally supported) avoids
	// sending the HELLO negotiation at all.
	opt.Protocol = 2

	return &Cache{rdb: redis.NewClient(opt)}, nil
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Cache) Close() error {
	return c.rdb.Close()
}

// Client exposes the underlying Redis client for callers that need
// Redis features beyond the cache-aside get/set above, such as the
// rate limiter's Lua script.
func (c *Cache) Client() *redis.Client {
	return c.rdb
}

func key(code string) string {
	return "link:" + code
}

// GetURL returns the cached long URL for a code, or ErrMiss if absent.
func (c *Cache) GetURL(ctx context.Context, code string) (string, error) {
	val, err := c.rdb.Get(ctx, key(code)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrMiss
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetURL populates the cache after a Postgres read (cache-aside fill).
func (c *Cache) SetURL(ctx context.Context, code, longURL string) error {
	return c.rdb.Set(ctx, key(code), longURL, linkTTL).Err()
}
