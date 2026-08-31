package yacymodel_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestParseHash(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"ABCDEFGHIJKL", true},
		{"____________", true},
		{"short", false},
		{"ABCDEFGHIJKLM", false},
		{"ABCDEFGHIJK=", false},
	}
	for _, c := range cases {
		_, err := yacymodel.ParseHash(c.in)
		if (err == nil) != c.ok {
			t.Errorf("ParseHash(%q) err = %v, want ok=%v", c.in, err, c.ok)
		}
		if err != nil && !errors.Is(err, yacymodel.ErrInvalidHash) {
			t.Errorf("ParseHash(%q) = %v, want ErrInvalidHash", c.in, err)
		}
	}
}

func TestHashBytesRoundTripThroughParseHashBytes(t *testing.T) {
	hash := yacymodel.WordHash("keyword")

	raw := hash.Bytes()
	if len(raw) != yacymodel.HashByteLength {
		t.Fatalf("Bytes length = %d, want %d", len(raw), yacymodel.HashByteLength)
	}

	parsed, err := yacymodel.ParseHashBytes(raw)
	if err != nil {
		t.Fatalf("ParseHashBytes(%x): %v", raw, err)
	}
	if parsed != hash {
		t.Errorf("ParseHashBytes = %q, want %q", parsed, hash)
	}
}

func TestParseHashBytesRejectsAnotherLength(t *testing.T) {
	tooFew := make([]byte, yacymodel.HashByteLength-1)

	if _, err := yacymodel.ParseHashBytes(tooFew); !errors.Is(err, yacymodel.ErrInvalidHash) {
		t.Errorf("ParseHashBytes = %v, want ErrInvalidHash", err)
	}
}

func mustParseHash(t *testing.T, s string) yacymodel.Hash {
	t.Helper()
	h, err := yacymodel.ParseHash(s)
	if err != nil {
		t.Fatalf("ParseHash(%q): %v", s, err)
	}
	return h
}

func TestWordHash(t *testing.T) {
	h := yacymodel.WordHash("Hello")
	if _, err := yacymodel.ParseHash(h.String()); err != nil {
		t.Fatalf("WordHash produced invalid hash %q: %v", h, err)
	}
	if len(h.String()) != yacymodel.HashLength {
		t.Errorf("WordHash length = %d, want %d", len(h.String()), yacymodel.HashLength)
	}
	if h != yacymodel.WordHash("hello") {
		t.Errorf("WordHash must lower-case: %q != %q", h, yacymodel.WordHash("hello"))
	}
	if yacymodel.WordHash("hello") == yacymodel.WordHash("world") {
		t.Error("distinct words must hash distinctly")
	}
}

func TestHashReserved(t *testing.T) {
	if !mustParseHash(t, "_____ABCDEFG").Reserved() {
		t.Error("expected reserved prefix to be detected")
	}
	if mustParseHash(t, "ABCDEFGHIJKL").Reserved() {
		t.Error("unexpected reserved detection")
	}
}
