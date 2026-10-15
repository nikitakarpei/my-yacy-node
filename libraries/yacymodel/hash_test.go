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
	if mustParseHash(t, "____-ABCDEFG").Reserved() {
		t.Error("unexpected reserved detection for four reserved symbols")
	}
}

func TestAHashOfTheFirstSymbolOnlyIsAHashAndNotTheZeroHash(t *testing.T) {
	first := mustParseHash(t, "AAAAAAAAAAAA")

	if first.IsZero() {
		t.Fatal("IsZero = true for a parsed hash, want false")
	}
	if first == (yacymodel.Hash{}) {
		t.Fatal("the parsed hash equals the zero hash")
	}
	if first.String() != "AAAAAAAAAAAA" {
		t.Fatalf("String = %q, want \"AAAAAAAAAAAA\"", first.String())
	}
}

func TestTheZeroHashHasNoTextAndNoBytes(t *testing.T) {
	var zero yacymodel.Hash

	if !zero.IsZero() || zero.String() != "" || zero.Bytes() != nil || zero.Reserved() {
		t.Fatalf("zero hash = %q, bytes %x, reserved %v, want no text, no bytes, not reserved",
			zero, zero.Bytes(), zero.Reserved())
	}
	if text, err := zero.MarshalText(); err != nil || len(text) != 0 {
		t.Fatalf("MarshalText = %q, %v, want no text", text, err)
	}
}

func TestAHashReadsBackFromItsText(t *testing.T) {
	for _, text := range []string{"ABCDEFGHIJKL", "zyx-_0129abc", "____________"} {
		if hash := mustParseHash(t, text); hash.String() != text {
			t.Fatalf("String = %q, want %q", hash.String(), text)
		}
	}
}

func TestHashBytesFollowWhatTheBufferHolds(t *testing.T) {
	hash := yacymodel.WordHash("keyword")

	appended := hash.AppendBytes([]byte("key:"))
	if string(appended[:4]) != "key:" || string(appended[4:]) != string(hash.Bytes()) {
		t.Fatalf("AppendBytes = %x, want \"key:\" then %x", appended, hash.Bytes())
	}
}

func TestHashesCompareInTheOrderOfTheirText(t *testing.T) {
	texts := []string{
		"ABCDEFGHIJKL",
		"aBCDEFGHIJKL",
		"0BCDEFGHIJKL",
		"-BCDEFGHIJKL",
		"_BCDEFGHIJKL",
		"ABCDEFGHIJK_",
	}
	for _, first := range texts {
		for _, second := range texts {
			want := yacymodel.CompareInAlphabetOrder(first, second)
			if got := mustParseHash(
				t,
				first,
			).CompareInAlphabetOrder(mustParseHash(t, second)); got != want {
				t.Fatalf("CompareInAlphabetOrder(%q, %q) = %d, want %d", first, second, got, want)
			}
		}
	}
	if (yacymodel.Hash{}).CompareInAlphabetOrder(mustParseHash(t, "AAAAAAAAAAAA")) >= 0 {
		t.Fatal("the zero hash does not sort before every hash")
	}
}
