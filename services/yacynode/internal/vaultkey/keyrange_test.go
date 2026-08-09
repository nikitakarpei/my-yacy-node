package vaultkey_test

import (
	"bytes"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestEveryKeyIsUnboundedOnBothSides(t *testing.T) {
	keys := vaultkey.EveryKey()

	if keys.FirstIncludedKey() != nil || keys.FirstExcludedKey() != nil {
		t.Fatalf("EveryKey = [%x, %x), want unbounded",
			keys.FirstIncludedKey(), keys.FirstExcludedKey())
	}
}

func TestContainsAnswersForEachSideOfBothBounds(t *testing.T) {
	keys := vaultkey.KeysUnder(vaultkey.KeyFrom([]byte("b")))

	for _, scenario := range []struct {
		key  string
		want bool
	}{
		{"a", false},
		{"b", true},
		{"bz", true},
		{"c", false},
		{"d", false},
	} {
		if got := keys.Contains([]byte(scenario.key)); got != scenario.want {
			t.Fatalf("Contains(%q) = %v, want %v", scenario.key, got, scenario.want)
		}
	}
}

func TestContainsHoldsEveryKeyAboveTheBoundOfAnUnboundedRange(t *testing.T) {
	keys := vaultkey.KeysFrom(vaultkey.KeyFrom([]byte("b")))

	if keys.Contains([]byte("a")) {
		t.Fatal("Contains holds a key below the included bound")
	}
	if !keys.Contains([]byte("b")) {
		t.Fatal("Contains excludes the included bound")
	}
	if !keys.Contains([]byte{0xFF, 0xFF}) {
		t.Fatal("Contains excludes the highest key of an unbounded range")
	}
}

func TestContainsHoldsNoKeyOfAnEmptyRange(t *testing.T) {
	keys := vaultkey.KeysBefore(vaultkey.KeyFrom([]byte{}))

	for _, key := range [][]byte{nil, {}, []byte("a"), {0xFF}} {
		if keys.Contains(key) {
			t.Fatalf("Contains(%x) holds a key of an empty range", key)
		}
	}
}

func TestKeysFromHoldsItsBoundAndIsUnboundedAbove(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Integer)
	keys := vaultkey.KeysFrom(layout.Key(10))

	if !keys.Contains(layout.Key(10).Bytes()) {
		t.Fatal("KeysFrom excludes its bound")
	}
	if keys.Contains(layout.Key(9).Bytes()) {
		t.Fatal("KeysFrom holds a smaller key")
	}
	if keys.FirstExcludedKey() != nil {
		t.Fatalf("FirstExcludedKey = %x, want nil", keys.FirstExcludedKey())
	}
}

func TestKeysUnderHoldsEveryKeyWithThePrefixAndNoOther(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.Text)
	keys := vaultkey.KeysUnder(layout.First("word"))

	for _, second := range orderedTexts() {
		if !keys.Contains(layout.Key("word", second).Bytes()) {
			t.Fatalf("KeysUnder excludes Key(%q, %q)", "word", second)
		}
	}

	for _, first := range []string{"wor", "word\x00", "words", "x"} {
		if keys.Contains(layout.Key(first, "").Bytes()) {
			t.Fatalf("KeysUnder holds Key(%q, %q)", first, "")
		}
	}
}

func TestKeysBeforeExcludesItsBound(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Integer)
	keys := vaultkey.KeysBefore(layout.Key(10))

	if keys.Contains(layout.Key(10).Bytes()) {
		t.Fatal("KeysBefore holds its bound")
	}
	if !keys.Contains(layout.Key(9).Bytes()) {
		t.Fatal("KeysBefore excludes a smaller key")
	}
}

func TestKeysThroughHoldsItsBoundAndEveryKeyUnderIt(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.Text)
	bound := layout.First("word")
	keys := vaultkey.KeysThrough(bound)

	if !keys.Contains(bound.Bytes()) {
		t.Fatal("KeysThrough excludes its bound")
	}
	if !keys.Contains(layout.Key("word", "\xff\xff").Bytes()) {
		t.Fatal("KeysThrough excludes a key under its bound")
	}
	if keys.Contains(layout.Key("words", "").Bytes()) {
		t.Fatal("KeysThrough holds a key past its bound")
	}
}

func TestUpperBoundTruncatesAtTheLastByteBelow0xFF(t *testing.T) {
	keys := vaultkey.KeysUnder(vaultkey.KeyFrom([]byte{0x01, 0xFF, 0xFF}))

	if want := []byte{0x02}; !bytes.Equal(keys.FirstExcludedKey(), want) {
		t.Fatalf("FirstExcludedKey = %x, want %x", keys.FirstExcludedKey(), want)
	}
}

func TestANilExcludedKeyMeansUnbounded(t *testing.T) {
	for _, prefix := range [][]byte{nil, {0xFF}, {0xFF, 0xFF}} {
		keys := vaultkey.KeysUnder(vaultkey.KeyFrom(prefix))
		if keys.FirstExcludedKey() != nil {
			t.Fatalf("KeysUnder(%x) upper bound = %x, want nil",
				prefix, keys.FirstExcludedKey())
		}
	}
}

func TestAnEmptyExcludedKeyDoesNotMeanUnbounded(t *testing.T) {
	keys := vaultkey.KeysBefore(vaultkey.KeyFrom([]byte{}))

	if keys.FirstExcludedKey() == nil {
		t.Fatal("KeysBefore an empty key reads as unbounded")
	}
	if len(keys.FirstExcludedKey()) != 0 {
		t.Fatalf("FirstExcludedKey = %x, want empty", keys.FirstExcludedKey())
	}
}
