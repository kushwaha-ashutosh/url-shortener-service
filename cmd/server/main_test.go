package main

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kushwaha-ashutosh/url-shortener/internal/cache"
)

func TestPingWithRetry_SucceedsImmediatelyWhenHealthy(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	c, err := cache.New(mr.Addr())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := pingWithRetry(context.Background(), c, 5, 10*time.Millisecond); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestPingWithRetry_RetriesThenFailsWhenUnreachable(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	addr := mr.Addr()
	mr.Close() // now nothing is listening at addr

	c, err := cache.New(addr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const attempts = 3
	const delay = 20 * time.Millisecond
	start := time.Now()

	err = pingWithRetry(context.Background(), c, attempts, delay)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error pinging an unreachable server")
	}
	// attempts-1 delays between attempts, e.g. 2 delays for 3 attempts.
	minExpected := delay * (attempts - 1)
	if elapsed < minExpected {
		t.Fatalf("expected at least %d retries (>= %v elapsed), got %v", attempts, minExpected, elapsed)
	}
}
