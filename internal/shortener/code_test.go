package shortener

import "testing"

func TestGenerate_Length(t *testing.T) {
	code, err := Generate(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 7 {
		t.Fatalf("expected length 7, got %d (%q)", len(code), code)
	}
}

func TestGenerate_NoAmbiguousChars(t *testing.T) {
	ambiguous := map[rune]bool{'0': true, 'O': true, '1': true, 'l': true, 'I': true}
	for i := 0; i < 1000; i++ {
		code, err := Generate(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range code {
			if ambiguous[r] {
				t.Fatalf("code %q contains ambiguous character %q", code, r)
			}
		}
	}
}

func TestGenerate_Distinct(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 500; i++ {
		code, err := Generate(8)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seen[code] {
			t.Fatalf("collision detected on code %q after %d draws", code, i)
		}
		seen[code] = true
	}
}
