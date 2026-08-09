package vaultkey_test

import (
	"bytes"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func containsKey(keys vaultkey.KeyRange, key vaultkey.Key) bool {
	if bytes.Compare(key.Bytes(), keys.FirstIncludedKey()) < 0 {
		return false
	}

	excluded := keys.FirstExcludedKey()

	return excluded == nil || bytes.Compare(key.Bytes(), excluded) < 0
}

func TestEveryKeyIsUnboundedOnBothSides(t *testing.T) {
	keys := vaultkey.EveryKey()

	if keys.FirstIncludedKey() != nil || keys.FirstExcludedKey() != nil {
		t.Fatalf("EveryKey = [%x, %x), want unbounded",
			keys.FirstIncludedKey(), keys.FirstExcludedKey())
	}
}

func TestKeysUnderHoldsEveryKeyWithThePrefixAndNoOther(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.Text)
	keys := vaultkey.KeysUnder(layout.First("word"))

	for _, second := range orderedTexts() {
		if !containsKey(keys, layout.Key("word", second)) {
			t.Fatalf("KeysUnder excludes Key(%q, %q)", "word", second)
		}
	}

	for _, first := range []string{"wor", "word\x00", "words", "x"} {
		if containsKey(keys, layout.Key(first, "")) {
			t.Fatalf("KeysUnder holds Key(%q, %q)", first, "")
		}
	}
}

func TestKeysBeforeExcludesItsBound(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Integer)
	keys := vaultkey.KeysBefore(layout.Key(10))

	if containsKey(keys, layout.Key(10)) {
		t.Fatal("KeysBefore holds its bound")
	}
	if !containsKey(keys, layout.Key(9)) {
		t.Fatal("KeysBefore excludes a smaller key")
	}
}

func TestKeysThroughHoldsItsBoundAndEveryKeyUnderIt(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.Text)
	bound := layout.First("word")
	keys := vaultkey.KeysThrough(bound)

	if !containsKey(keys, bound) {
		t.Fatal("KeysThrough excludes its bound")
	}
	if !containsKey(keys, layout.Key("word", "\xff\xff")) {
		t.Fatal("KeysThrough excludes a key under its bound")
	}
	if containsKey(keys, layout.Key("words", "")) {
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
