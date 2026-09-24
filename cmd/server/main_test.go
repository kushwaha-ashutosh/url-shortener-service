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

func TestRedisHost(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain host:port", "localhost:6379", "localhost"},
		{"redis URL", "redis://localhost:6379/0", "localhost"},
		{"rediss URL with auth", "rediss://default:secret@generous-mole-295263.upstash.io:6379", "generous-mole-295263.upstash.io"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redisHost(tc.input)
			if got != tc.want {
				t.Fatalf("redisHost(%q): got %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRedisAddr(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain host:port", "localhost:6379", "localhost:6379"},
		{"redis URL with port", "redis://localhost:6380/0", "localhost:6380"},
		{"rediss URL with auth and port", "rediss://default:secret@generous-mole-295263.upstash.io:6379", "generous-mole-295263.upstash.io:6379"},
		{"URL without explicit port defaults to 6379", "rediss://default:secret@example.upstash.io", "example.upstash.io:6379"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redisAddr(tc.input)
			if got != tc.want {
				t.Fatalf("redisAddr(%q): got %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
