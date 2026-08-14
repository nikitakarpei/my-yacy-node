package yacymodel_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestParseSeedListURL(t *testing.T) {
	u, err := yacymodel.ParseSeedListURL("https://example.org/seed.txt")
	if err != nil || u.String() != "https://example.org/seed.txt" {
		t.Fatalf("ParseSeedListURL = %q, %v", u.String(), err)
	}
}

func TestParseSeedListURLRoundTrips(t *testing.T) {
	for _, s := range []string{
		"https://example.org/seed.txt",
		"http://example.org:8090/yacy/seedlist.html?count=1000",
		"https://user@example.org/path/to/seed.txt#frag",
	} {
		u, err := yacymodel.ParseSeedListURL(s)
		if err != nil {
			t.Fatalf("ParseSeedListURL(%q): %v", s, err)
		}
		if u.String() != s {
			t.Errorf("ParseSeedListURL(%q).String() = %q", s, u.String())
		}
	}
}

func TestParseSeedListURLRejects(t *testing.T) {
	for _, s := range []string{"ftp://example.org/x", "https://", "://nope", "relative"} {
		if _, err := yacymodel.ParseSeedListURL(s); !errors.Is(err, yacymodel.ErrBadSeedListURL) {
			t.Fatalf("ParseSeedListURL(%q) = %v, want ErrBadSeedListURL", s, err)
		}
	}
}
