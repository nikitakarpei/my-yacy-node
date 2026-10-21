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
			func(text string, buffer []byte) []byte { return append(buffer, text...) },
			func(stored vault.StoredBytes) (string, error) { return string(stored.AppendTo(nil)), nil },
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

func TestTimeInSecondsKeysKeepTheStoredByteForm(t *testing.T) {
	for _, seconds := range []int64{0, 1_700_000_000, -1} {
		instant := time.Unix(seconds, 0)
		expectStoredByteForm(t, vault.SingleKey(vault.TimeInSecondsKeyPart).Key(instant), seconds)
		expectStoredByteForm(
			t,
			vault.SingleKey(vault.TimeInSecondsKeyPartDescending).Key(instant),
			orderedcode.Decr(seconds),
		)
	}
}

func TestTimeInMillisecondsKeysKeepTheStoredByteForm(t *testing.T) {
	for _, milliseconds := range []int64{0, 1_700_000_000_500, -1} {
		instant := time.UnixMilli(milliseconds)
		expectStoredByteForm(
			t,
			vault.SingleKey(vault.TimeInMillisecondsKeyPart).Key(instant),
			milliseconds,
		)
		expectStoredByteForm(
			t,
			vault.SingleKey(vault.TimeInMillisecondsKeyPartDescending).Key(instant),
			orderedcode.Decr(milliseconds),
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
		vault.TripleKey(vault.TimeInSecondsKeyPart, vault.TextKeyPart, vault.TextKeyPart).
			Key(instant, "word", "\xff"),
		instant.Unix(), "word", "\xff",
	)
	expectStoredByteForm(
		t,
		vault.QuadKey(vault.IntegerKeyPart, vault.TimeInSecondsKeyPart, vault.TextKeyPart, vault.TextKeyPart).
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

func TestKeysReadBackTheEdgeValuesTheyHold(t *testing.T) {
	for _, part := range []vault.KeyPart[string]{vault.TextKeyPart, vault.TextKeyPartDescending} {
		parts := vault.PairKey(part, vault.TextKeyPart)
		for _, text := range textsInKeys {
			first, second, err := parts.PartsOf(parts.Key(text, "after").Bytes())
			if err != nil || first != text || second != "after" {
				t.Fatalf("PartsOf(Key(%q)) = %q, %q, %v", text, first, second, err)
			}
		}
	}
	for _, part := range []vault.KeyPart[int64]{vault.IntegerKeyPart, vault.IntegerKeyPartDescending} {
		parts := vault.PairKey(part, vault.TextKeyPart)
		for _, integer := range integersInKeys() {
			first, second, err := parts.PartsOf(parts.Key(integer, "after").Bytes())
			if err != nil || first != integer || second != "after" {
				t.Fatalf("PartsOf(Key(%d)) = %d, %q, %v", integer, first, second, err)
			}
		}
	}
}

func TestPartsOfRejectsMalformedKeys(t *testing.T) {
	text := vault.SingleKey(vault.TextKeyPart)
	integer := vault.SingleKey(vault.IntegerKeyPart)
	textOfDomain := vault.SingleKey(vault.TextKeyPartFrom(
		func(text string) string { return text },
		func(text string) (string, error) { return text, nil },
	))
	bytesOfDomain := vault.SingleKey(vault.BytesKeyPartFrom(
		func(text string, buffer []byte) []byte { return append(buffer, text...) },
		func(stored vault.StoredBytes) (string, error) { return string(stored.AppendTo(nil)), nil },
	))
	for _, scenario := range []struct {
		name      string
		storedKey []byte
		partsOf   func([]byte) error
	}{
		{"UnterminatedText", []byte("word"), failureOfPartsOf(text)},
		{"TextWithABadEscape", []byte{'a', 0x00, 0x05}, failureOfPartsOf(text)},
		{"UnterminatedDomainText", []byte("word"), failureOfPartsOf(textOfDomain)},
		{"UnterminatedDomainBytes", []byte("word"), failureOfPartsOf(bytesOfDomain)},
		{"TrailingBytes", append(text.Key("word").Bytes(), 'x'), failureOfPartsOf(text)},
		{"MissingInteger", nil, failureOfPartsOf(integer)},
		{"IntegerShorterThanItsHeader", []byte{0xC0}, failureOfPartsOf(integer)},
		{"IntegerHeaderTooLong", []byte{0xFF, 0xFF, 0x00}, failureOfPartsOf(integer)},
		{"IntegerAboveInt64", []byte{0xFF, 0xC0, 0x80, 0, 0, 0, 0, 0, 0, 0}, failureOfPartsOf(integer)},
		{"IntegerWithBitsAboveInt64", []byte{0xFF, 0xDF, 0, 0, 0, 0, 0, 0, 0, 0}, failureOfPartsOf(integer)},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if err := scenario.partsOf(scenario.storedKey); err == nil {
				t.Fatalf("PartsOf(%x) succeeded, want an error", scenario.storedKey)
			}
		})
	}
}

func failureOfPartsOf[A any](parts vault.SingleKeyParts[A]) func([]byte) error {
	return func(storedKey []byte) error {
		_, err := parts.PartsOf(storedKey)

		return err
	}
}
