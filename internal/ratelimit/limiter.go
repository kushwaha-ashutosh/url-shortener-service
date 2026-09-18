// Package ratelimit implements a Redis-backed sliding-window rate
// limiter, shared across all instances of the API rather than kept
// in-process — the same limit applies whether there's one server or
// twenty behind a load balancer.
package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed sliding_window.lua
var slidingWindowScript string

type Limiter struct {
	rdb    *redis.Client
	script *redis.Script
	limit  int
	window time.Duration
}

func New(rdb *redis.Client, limit int, window time.Duration) *Limiter {
	return &Limiter{
		rdb:    rdb,
		script: redis.NewScript(slidingWindowScript),
		limit:  limit,
		window: window,
	}
}

// Allow reports whether a request identified by key is within the
// configured limit for the current window. On a Redis error, the
// caller decides whether to fail open or closed — this just surfaces
// the error rather than making that call itself.
func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now().UnixNano()
	member := fmt.Sprintf("%d-%d", now, rand.Int63())

	res, err := l.script.Run(ctx, l.rdb,
		[]string{"ratelimit:" + key},
		now, l.window.Nanoseconds(), l.limit, member,
	).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}
