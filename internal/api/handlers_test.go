package api

import (
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid https", "https://example.com/page", false},
		{"valid http", "http://example.com", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"javascript scheme rejected", "javascript:alert(1)", true},
		{"data scheme rejected", "data:text/html,<script>alert(1)</script>", true},
		{"file scheme rejected", "file:///etc/passwd", true},
		{"missing host", "https://", true},
		{"missing scheme", "example.com", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := normalizeURL(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("normalizeURL(%q): got err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestValidateCustomCode(t *testing.T) {
	cases := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid alnum", "my-link_1", false},
		{"valid minimum length", "abc", false},
		{"valid maximum length", strings.Repeat("a", 32), false},
		{"too short", "ab", true},
		{"too long", strings.Repeat("a", 33), true},
		{"empty", "", true},
		{"contains slash", "a/b", true},
		{"contains space", "a b", true},
		{"reserved api", "api", true},
		{"reserved healthz", "healthz", true},
		{"reserved case insensitive", "HealthZ", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCustomCode(tc.code)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateCustomCode(%q): got err=%v, wantErr=%v", tc.code, err, tc.wantErr)
			}
		})
	}
}

func TestShortURLFor(t *testing.T) {
	got := shortURLFor("http://localhost:8081", "abc123")
	want := "http://localhost:8081/abc123"
	if got != want {
		t.Fatalf("shortURLFor: got %q, want %q", got, want)
	}
}
