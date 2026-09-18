package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestLimiter(t *testing.T, limit int, window time.Duration) *Limiter {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return New(rdb, limit, window)
}

func TestLimiter_AllowsUpToLimit(t *testing.T) {
	l := newTestLimiter(t, 3, time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		allowed, err := l.Allow(ctx, "1.2.3.4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d: expected allowed, got denied", i)
		}
	}
}

func TestLimiter_DeniesOverLimit(t *testing.T) {
	l := newTestLimiter(t, 3, time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := l.Allow(ctx, "1.2.3.4"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	allowed, err := l.Allow(ctx, "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("expected 4th request within the window to be denied")
	}
}

func TestLimiter_KeysAreIndependent(t *testing.T) {
	l := newTestLimiter(t, 1, time.Minute)
	ctx := context.Background()

	if allowed, err := l.Allow(ctx, "client-a"); err != nil || !allowed {
		t.Fatalf("client-a first request: allowed=%v err=%v", allowed, err)
	}
	if allowed, err := l.Allow(ctx, "client-a"); err != nil || allowed {
		t.Fatalf("client-a second request: expected denied, allowed=%v err=%v", allowed, err)
	}
	if allowed, err := l.Allow(ctx, "client-b"); err != nil || !allowed {
		t.Fatalf("client-b first request: expected allowed, allowed=%v err=%v", allowed, err)
	}
}

func TestLimiter_ConcurrentRequestsNeverExceedLimit(t *testing.T) {
	const limit = 20
	l := newTestLimiter(t, limit, time.Minute)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := l.Allow(ctx, "burst")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowedCount != limit {
		t.Fatalf("expected exactly %d requests admitted under concurrent burst, got %d", limit, allowedCount)
	}
}
