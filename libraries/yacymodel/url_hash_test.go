package yacymodel_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestURLHashIsAParseableTwelveSymbolHash(t *testing.T) {
	hash := hashOfAddress(t, "http://example.com/path?q=1")
	if len(hash.String()) != yacymodel.HashLength {
		t.Fatalf("url hash length = %d, want %d", len(hash.String()), yacymodel.HashLength)
	}
	if _, err := yacymodel.ParseURLHash(hash.String()); err != nil {
		t.Errorf("url hash %q does not parse: %v", hash, err)
	}
}

func TestURLHashBytesRoundTripThroughParseURLHashBytes(t *testing.T) {
	hash := hashOfAddress(t, "http://example.com/path?q=1")

	parsed, err := yacymodel.ParseURLHashBytes(hash.Bytes())
	if err != nil {
		t.Fatalf("ParseURLHashBytes(%x): %v", hash.Bytes(), err)
	}
	if parsed != hash {
		t.Errorf("ParseURLHashBytes = %q, want %q", parsed, hash)
	}
}

func TestParseURLHashBytesRejectsAnotherLength(t *testing.T) {
	tooFew := make([]byte, yacymodel.HashByteLength-1)

	if _, err := yacymodel.ParseURLHashBytes(tooFew); !errors.Is(err, yacymodel.ErrInvalidHash) {
		t.Errorf("ParseURLHashBytes = %v, want ErrInvalidHash", err)
	}
}

func TestURLHashIsDeterministic(t *testing.T) {
	first := hashOfAddress(t, "http://example.com/")
	second := hashOfAddress(t, "http://example.com/")
	if first != second {
		t.Errorf("url hash must be deterministic: %q then %q", first, second)
	}
}

func TestURLHashMatchesRealYaCyForSpecialUseTLD(t *testing.T) {
	const (
		address = "http://transfer.example.invalid/doc-150.txt"
		want    = "TULSZfg4-80c"
	)
	if got := hashOfAddress(t, address); got.String() != want {
		t.Errorf("hash of %q = %q, want %q", address, got, want)
	}
}

func TestURLHashEncodingIsFrozen(t *testing.T) {
	cases := map[string]string{
		"http://example.de/": "-QU1M7NH-7-A",
		"http://example.ru/": "CYe0z7P0yZBA",
		"http://example.br/": "3kp2X7J0kuTE",
		"http://example.jp/": "Re7-j7TLc0dI",
		"http://example.ir/": "LU5jE7lSmaAM",
		"http://example.us/": "pug_Z7L2Q4LQ",
		"http://example.za/": "gt1MV7E2yesU",

		"http://example.com/": "pr8XV7QpK89Y",
		"http://example.zip/": "t3W5Y7JLp4kY",
		"http://8.8.8.8/":     "CBPqTsg_XLIY",

		"http://localhost/":     "ydtWn7sIW_pc",
		"http://127.0.0.1/":     "aZ_PRjJhfryc",
		"http://site.onion/":    "cgKOV7BYHCjc",
		"http://printer.local/": "29i2X7wOGGwc",
		"http://doc.example/":   "q2iRT7ZE1a9c",

		"http://exampleab.com/":         "AMOmD7IlHtxZ",
		"http://longexampledomain.com/": "zMNQE7pM_BVb",
		"https://example.com/":          "GCzO2s5gBFP4",
		"http://bücher.de/":             "0Vuap7fMfOoC",
	}
	for address, want := range cases {
		if got := hashOfAddress(t, address); got.String() != want {
			t.Errorf("hash of %q = %q, want %q", address, got, want)
		}
	}
}

func TestURLHashIgnoresTheHostCase(t *testing.T) {
	lower := hashOfAddress(t, "http://example.com/Path")
	mixed := hashOfAddress(t, "http://Example.COM/Path")
	if lower != mixed {
		t.Errorf("a mixed-case host must not change the url hash: %q vs %q", mixed, lower)
	}
}

func TestURLHashDistinguishesPathsOnOneHost(t *testing.T) {
	first := hashOfAddress(t, "http://example.com/one")
	second := hashOfAddress(t, "http://example.com/two")
	if first == second {
		t.Errorf("different paths must yield different url hashes, both %q", first)
	}
}

func TestURLHashDistinguishesASubdomain(t *testing.T) {
	bare := hashOfAddress(t, "http://example.com/")
	sub := hashOfAddress(t, "http://www.example.com/")
	if bare == sub {
		t.Errorf("a subdomain must change the url hash, both %q", bare)
	}
}

func TestURLHashIgnoresDotSegments(t *testing.T) {
	dotted := hashOfAddress(t, "http://example.com/a/b/../c")
	resolved := hashOfAddress(t, "http://example.com/a/c")
	if dotted != resolved {
		t.Errorf("dot-segment url hash %q must equal resolved %q", dotted, resolved)
	}
}

func TestURLHashUsesThePunycodeHost(t *testing.T) {
	unicode := hashOfAddress(t, "http://bücher.example/")
	ascii := hashOfAddress(t, "http://xn--bcher-kva.example/")
	if unicode != ascii {
		t.Errorf("idn url hash %q must equal punycode form %q", unicode, ascii)
	}
}

func hashOfAddress(t *testing.T, raw string) yacymodel.URLHash {
	t.Helper()

	return normalformOfAddress(t, raw).Hash()
}
