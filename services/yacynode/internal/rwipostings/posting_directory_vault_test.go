package rwipostings_test

import (
	"errors"
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

func mustLanguage(t *testing.T, raw string) yacymodel.Language {
	t.Helper()

	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		t.Fatalf("parse language %q: %v", raw, err)
	}

	return language
}

func fullPosting(t *testing.T) yacymodel.RWIPosting {
	t.Helper()

	return yacymodel.RWIPosting{
		LastModified:  yacymodel.MicroDate(19876),
		TitleWords:    3,
		TextWords:     258,
		Phrases:       7,
		DocumentType:  yacymodel.DocumentTypeHTML,
		Language:      mustLanguage(t, "en"),
		LocalLinks:    4,
		ExternalLinks: 2,
		URLLength:     41,
		URLComponents: 5,
		Appearance: yacymodel.Appearance{
			AppearsInTitle: true,
			HasImage:       true,
			Emphasized:     true,
		},
		Hits:                   1,
		TextPosition:           512,
		PhraseRelativePosition: 9,
		PhrasePosition:         11,
	}
}

func TestStoredPostingRoundTrip(t *testing.T) {
	codec := rwipostings.PostingCodec()

	entry := fullPosting(t)

	encoded, err := codec.Encode(entry)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded != entry {
		t.Errorf("round trip =\n %+v\nwant\n %+v", decoded, entry)
	}
}

func TestStoredPostingRoundTripsCardinalsAtTheEdgesOfTheirWidths(t *testing.T) {
	codec := rwipostings.PostingCodec()

	for name, written := range map[string]yacymodel.RWIPosting{
		"zero":    postingFromCardinals(t, 0, 0),
		"ceiling": postingFromCardinals(t, math.MaxUint8, math.MaxUint16),
	} {
		encoded, err := codec.Encode(written)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
		decoded, err := codec.Decode(encoded)
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if decoded != written {
			t.Errorf("%s round trips to\n %+v\nwant\n %+v", name, decoded, written)
		}
	}
}

func postingFromCardinals(
	t *testing.T,
	oneByteCardinal uint8,
	twoByteCardinal uint16,
) yacymodel.RWIPosting {
	t.Helper()

	return yacymodel.RWIPosting{
		LastModified:           yacymodel.MicroDate(twoByteCardinal),
		Language:               mustLanguage(t, "en"),
		TitleWords:             oneByteCardinal,
		TextWords:              twoByteCardinal,
		Phrases:                twoByteCardinal,
		LocalLinks:             oneByteCardinal,
		ExternalLinks:          oneByteCardinal,
		URLLength:              oneByteCardinal,
		URLComponents:          oneByteCardinal,
		Hits:                   oneByteCardinal,
		TextPosition:           twoByteCardinal,
		PhraseRelativePosition: oneByteCardinal,
		PhrasePosition:         oneByteCardinal,
	}
}

func TestStoredPostingWrapsOneByteAndSaturatesTwoByteCardinals(t *testing.T) {
	decoded := decodedPosting(t, storedPostingOf(20000, 300, 70000))

	want := postingFromCardinals(t, 300%256, math.MaxUint16)
	want.LastModified = 20000
	if decoded != want {
		t.Errorf("decoded =\n %+v\nwant\n %+v", decoded, want)
	}
}

func storedPostingOf(lastModifiedDays int64, oneByteCardinal, twoByteCardinal int) []byte {
	var stored storedfields.Writer
	stored.Varint(lastModifiedDays)
	stored.Count(oneByteCardinal)
	stored.Count(twoByteCardinal)
	stored.Count(twoByteCardinal)
	stored.Byte(0)
	stored.Text("en")
	for range 4 {
		stored.Count(oneByteCardinal)
	}
	stored.Uint16(0)
	stored.Count(oneByteCardinal)
	stored.Count(twoByteCardinal)
	stored.Count(oneByteCardinal)
	stored.Count(oneByteCardinal)

	return stored.Record()
}

func decodedPosting(t *testing.T, stored []byte) yacymodel.RWIPosting {
	t.Helper()

	decoded, err := rwipostings.PostingCodec().Decode(stored)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	return decoded
}

func TestStoredPostingReadsADayOutsideTwoBytesAsNoDay(t *testing.T) {
	for _, days := range []int64{-1, math.MaxUint16 + 1} {
		if got := decodedPosting(t, storedPostingOf(days, 0, 0)).LastModified; got != 0 {
			t.Errorf("last modified of %d stored days = %d, want 0", days, got)
		}
	}
}

func TestStoredPostingRejectsEmptyValue(t *testing.T) {
	codec := rwipostings.PostingCodec()

	if _, err := codec.Decode(nil); !errors.Is(err, yacymodel.ErrBadRWIPosting) {
		t.Errorf("err = %v, want ErrBadRWIPosting", err)
	}
}

func TestStoredPostingRejectsAValueCutInsideAField(t *testing.T) {
	codec := rwipostings.PostingCodec()

	cutInsideTheLastModifiedVarint := []byte{0x80}
	if _, err := codec.Decode(cutInsideTheLastModifiedVarint); !errors.Is(
		err, yacymodel.ErrBadRWIPosting,
	) {
		t.Errorf("err = %v, want ErrBadRWIPosting", err)
	}
}
