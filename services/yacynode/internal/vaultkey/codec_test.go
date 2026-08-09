package vaultkey_test

import (
	"bytes"
	"math"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func orderedTexts() []string {
	return []string{
		"", "\x00", "\x00\x01", "\x00\xff", "\x01", "a", "ab", "b", "\xff", "\xff\x00", "\xff\xff",
	}
}

func orderedIntegers() []int64 {
	return []int64{math.MinInt64, -(1 << 40), -256, -1, 0, 1, 255, 1 << 40, math.MaxInt64}
}

func orderedInstants() []time.Time {
	return []time.Time{
		{},
		time.Date(1500, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Unix(-(1 << 31), 0),
		time.Unix(-1, 999999999),
		time.Unix(0, 0),
		time.Unix(0, 1),
		time.Unix(1, 0),
		time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC),
		time.Unix(0, math.MaxInt64),
		time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
}

func assertKeysSortAscending[A any](t *testing.T, keyOf func(A) vaultkey.Key, ordered []A) {
	t.Helper()

	for position := 1; position < len(ordered); position++ {
		earlier, later := ordered[position-1], ordered[position]
		if bytes.Compare(keyOf(earlier).Bytes(), keyOf(later).Bytes()) >= 0 {
			t.Fatalf("key(%v) does not sort before key(%v)", earlier, later)
		}
	}
}

func assertKeysSortDescending[A any](t *testing.T, keyOf func(A) vaultkey.Key, ordered []A) {
	t.Helper()

	for position := 1; position < len(ordered); position++ {
		earlier, later := ordered[position-1], ordered[position]
		if bytes.Compare(keyOf(earlier).Bytes(), keyOf(later).Bytes()) <= 0 {
			t.Fatalf("key(%v) does not sort after key(%v)", earlier, later)
		}
	}
}

func TestTextKeysSortInTextOrder(t *testing.T) {
	assertKeysSortAscending(t, vaultkey.Single(vaultkey.Text).Key, orderedTexts())
	assertKeysSortDescending(t, vaultkey.Single(vaultkey.TextDescending).Key, orderedTexts())
}

func TestIntegerKeysSortInNumericOrder(t *testing.T) {
	assertKeysSortAscending(t, vaultkey.Single(vaultkey.Integer).Key, orderedIntegers())
	assertKeysSortDescending(t, vaultkey.Single(vaultkey.IntegerDescending).Key, orderedIntegers())
}

func TestTimeKeysSortInChronologicalOrder(t *testing.T) {
	assertKeysSortAscending(t, vaultkey.Single(vaultkey.Time).Key, orderedInstants())
	assertKeysSortDescending(t, vaultkey.Single(vaultkey.TimeDescending).Key, orderedInstants())
}

func TestTextRoundTripsThroughBothDirections(t *testing.T) {
	for _, text := range orderedTexts() {
		for _, layout := range []vaultkey.SingleKey[string]{
			vaultkey.Single(vaultkey.Text),
			vaultkey.Single(vaultkey.TextDescending),
		} {
			decoded, err := layout.Parts(layout.Key(text))
			if err != nil {
				t.Fatalf("Parts(%q) failed: %v", text, err)
			}
			if decoded != text {
				t.Fatalf("Parts = %q, want %q", decoded, text)
			}
		}
	}
}

func TestIntegerRoundTripsThroughBothDirections(t *testing.T) {
	for _, number := range orderedIntegers() {
		for _, layout := range []vaultkey.SingleKey[int64]{
			vaultkey.Single(vaultkey.Integer),
			vaultkey.Single(vaultkey.IntegerDescending),
		} {
			decoded, err := layout.Parts(layout.Key(number))
			if err != nil {
				t.Fatalf("Parts(%d) failed: %v", number, err)
			}
			if decoded != number {
				t.Fatalf("Parts = %d, want %d", decoded, number)
			}
		}
	}
}

func TestTimeRoundTripsToTheSameNanosecondInUTC(t *testing.T) {
	for _, instant := range orderedInstants() {
		for _, layout := range []vaultkey.SingleKey[time.Time]{
			vaultkey.Single(vaultkey.Time),
			vaultkey.Single(vaultkey.TimeDescending),
		} {
			decoded, err := layout.Parts(layout.Key(instant))
			if err != nil {
				t.Fatalf("Parts(%s) failed: %v", instant, err)
			}
			if !decoded.Equal(instant) {
				t.Fatalf("Parts = %s, want %s", decoded, instant)
			}
			if decoded.Location() != time.UTC {
				t.Fatalf("Parts location = %s, want UTC", decoded.Location())
			}
		}
	}
}

func TestTimeDropsTheMonotonicReadingAndTheLocation(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Time)
	zone := time.FixedZone("somewhere", 3600)
	instant := time.Now().In(zone)

	decoded, err := layout.Parts(layout.Key(instant))
	if err != nil {
		t.Fatalf("Parts failed: %v", err)
	}

	if !decoded.Equal(instant) {
		t.Fatalf("Parts = %s, want %s", decoded, instant)
	}
	if decoded.Location() != time.UTC {
		t.Fatalf("Parts location = %s, want UTC", decoded.Location())
	}
	if decoded != decoded.Round(0) {
		t.Fatalf("Parts kept a monotonic reading: %s", decoded)
	}
}
