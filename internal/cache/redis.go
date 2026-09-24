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

// Tight timeouts and no retries: Redis sits on the redirect hot path as
// an optional accelerator, with Postgres as the real source of truth.
// go-redis's defaults (5s dial / 3s read+write, 3 retries with backoff)
// let a single request hang for 20-30+ seconds when Redis is down
// before the cache-aside fallback ever gets a chance to run — found by
// actually killing Redis under load rather than trusting the fallback
// existed on paper. Fail fast instead, so a Redis outage costs a bounded
// ~750ms worst case, not tens of seconds, before falling through.
const (
	dialTimeout  = 250 * time.Millisecond
	readTimeout  = 250 * time.Millisecond
	writeTimeout = 250 * time.Millisecond
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
