package yacymodel

import (
	"errors"
	"testing"
)

func TestParseSeedListURL(t *testing.T) {
	u, err := ParseSeedListURL("https://example.org/seed.txt")
	if err != nil || u.String() != "https://example.org/seed.txt" {
		t.Fatalf("ParseSeedListURL = %q, %v", u.String(), err)
	}
}

func TestParseSeedListURLRejects(t *testing.T) {
	for _, s := range []string{"ftp://example.org/x", "https://", "://nope", "relative"} {
		if _, err := ParseSeedListURL(s); !errors.Is(err, ErrBadSeedListURL) {
			t.Fatalf("ParseSeedListURL(%q) = %v, want ErrBadSeedListURL", s, err)
		}
	}
}
