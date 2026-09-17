package api

import "testing"

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
