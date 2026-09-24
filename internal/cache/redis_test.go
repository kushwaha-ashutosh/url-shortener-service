package cache

import "testing"

func TestNew_AcceptsPlainAddr(t *testing.T) {
	c, err := New("localhost:6379")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected a non-nil cache")
	}
}

func TestNew_AcceptsRedisURL(t *testing.T) {
	c, err := New("redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected a non-nil cache")
	}
}

func TestNew_AcceptsTLSRedisURL(t *testing.T) {
	// The rediss:// scheme (TLS) is what a managed provider like
	// Upstash requires in production, unlike the plain redis:// used
	// for local Docker Compose.
	c, err := New("rediss://default:password@example.upstash.io:6379")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected a non-nil cache")
	}
}

func TestNew_RejectsInvalidURL(t *testing.T) {
	_, err := New("redis://%zz")
	if err == nil {
		t.Fatal("expected an error for a malformed redis URL")
	}
}
