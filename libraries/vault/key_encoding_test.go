package vault_test

import (
	"bytes"
	"math"
	"testing"
	"time"

	"github.com/google/orderedcode"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var textsInKeys = []string{
	"", "a", "word", "\x00", "\xff", "a\x00b", "a\xffb", "\x00\xff\x00", "\xff\xff", "\x00\x01",
}

var edgeIntegersInKeys = []int64{0, 1, -1, 63, 64, -64, -65, math.MaxInt64, math.MinInt64}

const integersBesidePowerOfTwo = 6

func integersInKeys() []int64 {
	integers := make([]int64, 0, len(edgeIntegersInKeys)+integersBesidePowerOfTwo*63)
	integers = append(integers, edgeIntegersInKeys...)
	for shift := range 63 {
		power := int64(1) << shift
		integers = append(integers, power-1, power, power+1, -power-1, -power, -power+1)
	}

	return integers
}

func TestTextKeysKeepTheStoredByteForm(t *testing.T) {
	for _, text := range textsInKeys {
		expectStoredByteForm(t, vault.SingleKey(vault.TextKeyPart).Key(text), text)
		expectStoredByteForm(
			t,
			vault.SingleKey(vault.TextKeyPartDescending).Key(text),
			orderedcode.Decr(text),
		)
		bytesPart := vault.BytesKeyPartFrom(
			func(text string) []byte { return []byte(text) },
			func(stored []byte) (string, error) { return string(stored), nil },
		)
		expectStoredByteForm(t, vault.SingleKey(bytesPart).Key(text), text)
	}
}

func TestIntegerKeysKeepTheStoredByteForm(t *testing.T) {
	for _, integer := range integersInKeys() {
		expectStoredByteForm(t, vault.SingleKey(vault.IntegerKeyPart).Key(integer), integer)
		expectStoredByteForm(
			t,
			vault.SingleKey(vault.IntegerKeyPartDescending).Key(integer),
			orderedcode.Decr(integer),
		)
	}
}

func TestTimeKeysKeepTheStoredByteForm(t *testing.T) {
	for _, seconds := range []int64{0, 1_700_000_000, -1} {
		instant := time.Unix(seconds, 0)
		expectStoredByteForm(t, vault.SingleKey(vault.TimeKeyPart).Key(instant), seconds)
		expectStoredByteForm(
			t,
			vault.SingleKey(vault.TimeKeyPartDescending).Key(instant),
			orderedcode.Decr(seconds),
		)
	}
}

func TestKeysOfSeveralPartsKeepTheStoredByteForm(t *testing.T) {
	instant := time.Unix(1_700_000_000, 0)
	expectStoredByteForm(
		t,
		vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPartDescending).Key(-300, "a\x00b"),
		int64(-300), orderedcode.Decr("a\x00b"),
	)
	expectStoredByteForm(
		t,
		vault.TripleKey(vault.TimeKeyPart, vault.TextKeyPart, vault.TextKeyPart).
			Key(instant, "word", "\xff"),
		instant.Unix(), "word", "\xff",
	)
	expectStoredByteForm(
		t,
		vault.QuadKey(vault.IntegerKeyPart, vault.TimeKeyPart, vault.TextKeyPart, vault.TextKeyPart).
			Key(7, instant, "word", "url"),
		int64(7),
		instant.Unix(),
		"word",
		"url",
	)
}

func expectStoredByteForm(t *testing.T, key vault.Key, storedItems ...any) {
	t.Helper()

	stored, err := orderedcode.Append(nil, storedItems...)
	if err != nil {
		t.Fatalf("orderedcode.Append(%v): %v", storedItems, err)
	}
	if !bytes.Equal(key.Bytes(), stored) {
		t.Fatalf("key of %v = %x, want the stored byte form %x", storedItems, key.Bytes(), stored)
	}
}
