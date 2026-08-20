package vault_test

import (
	"bytes"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
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

type documentLanguage struct {
	tag string
}

func documentLanguageTagOf(language documentLanguage) string {
	return language.tag
}

var errBlankLanguageTag = errors.New("language tag is blank")

func documentLanguageFrom(tag string) (documentLanguage, error) {
	if tag == "" {
		return documentLanguage{}, errBlankLanguageTag
	}

	return documentLanguage{tag: tag}, nil
}

func orderedDocumentLanguages() []documentLanguage {
	return []documentLanguage{
		{tag: "\x00"}, {tag: "\x01"}, {tag: "de"}, {tag: "en"}, {tag: "eng"}, {tag: "\xff"},
	}
}

func assertKeysSortAscending[A any](t *testing.T, keyOf func(A) vault.Key, ordered []A) {
	t.Helper()

	for position := 1; position < len(ordered); position++ {
		earlier, later := ordered[position-1], ordered[position]
		if bytes.Compare(keyOf(earlier).Bytes(), keyOf(later).Bytes()) >= 0 {
			t.Fatalf("key(%v) does not sort before key(%v)", earlier, later)
		}
	}
}

func assertKeysSortDescending[A any](t *testing.T, keyOf func(A) vault.Key, ordered []A) {
	t.Helper()

	for position := 1; position < len(ordered); position++ {
		earlier, later := ordered[position-1], ordered[position]
		if bytes.Compare(keyOf(earlier).Bytes(), keyOf(later).Bytes()) <= 0 {
			t.Fatalf("key(%v) does not sort after key(%v)", earlier, later)
		}
	}
}

func TestTextKeysSortInTextOrder(t *testing.T) {
	assertKeysSortAscending(t, vault.SingleKey(vault.TextKeyPart).Key, orderedTexts())
	assertKeysSortDescending(t, vault.SingleKey(vault.TextKeyPartDescending).Key, orderedTexts())
}

func TestIntegerKeysSortInNumericOrder(t *testing.T) {
	assertKeysSortAscending(t, vault.SingleKey(vault.IntegerKeyPart).Key, orderedIntegers())
	assertKeysSortDescending(
		t,
		vault.SingleKey(vault.IntegerKeyPartDescending).Key,
		orderedIntegers(),
	)
}

func TestTimeKeysSortInChronologicalOrder(t *testing.T) {
	assertKeysSortAscending(t, vault.SingleKey(vault.TimeKeyPart).Key, orderedInstants())
	assertKeysSortDescending(t, vault.SingleKey(vault.TimeKeyPartDescending).Key, orderedInstants())
}

func TestTextRoundTripsThroughBothDirections(t *testing.T) {
	for _, text := range orderedTexts() {
		for _, layout := range []vault.SingleKeyLayout[string]{
			vault.SingleKey(vault.TextKeyPart),
			vault.SingleKey(vault.TextKeyPartDescending),
		} {
			decoded, err := layout.Parts(layout.Key(text).Bytes())
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
		for _, layout := range []vault.SingleKeyLayout[int64]{
			vault.SingleKey(vault.IntegerKeyPart),
			vault.SingleKey(vault.IntegerKeyPartDescending),
		} {
			decoded, err := layout.Parts(layout.Key(number).Bytes())
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
		for _, layout := range []vault.SingleKeyLayout[time.Time]{
			vault.SingleKey(vault.TimeKeyPart),
			vault.SingleKey(vault.TimeKeyPartDescending),
		} {
			decoded, err := layout.Parts(layout.Key(instant).Bytes())
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
	layout := vault.SingleKey(vault.TimeKeyPart)
	zone := time.FixedZone("somewhere", 3600)
	instant := time.Now().In(zone)

	decoded, err := layout.Parts(layout.Key(instant).Bytes())
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

func TestTextKeyPartFromKeysSortInTextOrder(t *testing.T) {
	language := vault.TextKeyPartFrom(documentLanguageTagOf, documentLanguageFrom)
	assertKeysSortAscending(t, vault.SingleKey(language).Key, orderedDocumentLanguages())

	text := vault.SingleKey(vault.TextKeyPart)
	for _, decoded := range orderedDocumentLanguages() {
		if !bytes.Equal(
			vault.SingleKey(language).Key(decoded).Bytes(),
			text.Key(decoded.tag).Bytes(),
		) {
			t.Fatalf("key(%v) differs from the text key of %q", decoded, decoded.tag)
		}
	}
}

func TestTextKeyPartFromRoundTripsThroughTheDomainValue(t *testing.T) {
	layout := vault.SingleKey(vault.TextKeyPartFrom(documentLanguageTagOf, documentLanguageFrom))

	for _, language := range orderedDocumentLanguages() {
		decoded, err := layout.Parts(layout.Key(language).Bytes())
		if err != nil {
			t.Fatalf("Parts(%v) failed: %v", language, err)
		}
		if decoded != language {
			t.Fatalf("Parts = %v, want %v", decoded, language)
		}
	}
}

func TestTextKeyPartFromFailsWhenTheTextIsNoDomainValue(t *testing.T) {
	language := vault.TextKeyPartFrom(documentLanguageTagOf, documentLanguageFrom)
	pairOfTexts := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart)
	tripleOfTexts := vault.TripleKey(vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart)

	partsFailures := map[string]func() error{
		"single": func() error {
			_, err := vault.SingleKey(language).
				Parts(vault.SingleKey(vault.TextKeyPart).Key("").Bytes())

			return err
		},
		"pair first": func() error {
			_, _, err := vault.PairKey(language, vault.TextKeyPart).
				Parts(pairOfTexts.Key("", "en").Bytes())

			return err
		},
		"pair second": func() error {
			_, _, err := vault.PairKey(vault.TextKeyPart, language).
				Parts(pairOfTexts.Key("en", "").Bytes())

			return err
		},
		"triple first": func() error {
			layout := vault.TripleKey(language, vault.TextKeyPart, vault.TextKeyPart)
			_, _, _, err := layout.Parts(tripleOfTexts.Key("", "en", "en").Bytes())

			return err
		},
		"triple second": func() error {
			layout := vault.TripleKey(vault.TextKeyPart, language, vault.TextKeyPart)
			_, _, _, err := layout.Parts(tripleOfTexts.Key("en", "", "en").Bytes())

			return err
		},
		"triple third": func() error {
			layout := vault.TripleKey(vault.TextKeyPart, vault.TextKeyPart, language)
			_, _, _, err := layout.Parts(tripleOfTexts.Key("en", "en", "").Bytes())

			return err
		},
	}

	for position, partsFailure := range partsFailures {
		t.Run(position, func(t *testing.T) {
			if err := partsFailure(); !errors.Is(err, errBlankLanguageTag) {
				t.Fatalf("Parts error = %v, want %v", err, errBlankLanguageTag)
			}
		})
	}
}
