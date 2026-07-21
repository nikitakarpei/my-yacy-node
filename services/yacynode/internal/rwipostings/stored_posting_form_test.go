package rwipostings

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func urlHashFromWord(t *testing.T, word string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(word).String())
	if err != nil {
		t.Fatalf("url hash for %q: %v", word, err)
	}

	return hash
}

func mustLanguage(t *testing.T, raw string) yacymodel.Optional[yacymodel.Language] {
	t.Helper()

	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		t.Fatalf("parse language %q: %v", raw, err)
	}

	return yacymodel.Some(language)
}

func fullPosting(t *testing.T) yacymodel.RWIPosting {
	t.Helper()

	return yacymodel.RWIPosting{
		URLHash:       urlHashFromWord(t, "u1"),
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
	entry := fullPosting(t)

	encoded, err := postingCodec{}.Encode(entry)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := postingCodec{}.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded != entry {
		t.Errorf("round trip =\n %+v\nwant\n %+v", decoded, entry)
	}
}

func TestStoredPostingRejectsEmptyValue(t *testing.T) {
	if _, err := (postingCodec{}).Decode(nil); !errors.Is(err, yacymodel.ErrBadRWIPosting) {
		t.Errorf("err = %v, want ErrBadRWIPosting", err)
	}
}

func TestStoredPostingRejectsUnknownFormat(t *testing.T) {
	encoded, err := postingCodec{}.Encode(fullPosting(t))
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	encoded[0] = 0x01

	if _, err := (postingCodec{}).Decode(encoded); !errors.Is(err, yacymodel.ErrBadRWIPosting) {
		t.Errorf("err = %v, want ErrBadRWIPosting for old 0x01 blob", err)
	}
}

func TestStoredPostingRejectsTruncatedBinary(t *testing.T) {
	encoded, err := postingCodec{}.Encode(fullPosting(t))
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	for length := 1; length < len(encoded); length++ {
		if _, err := (postingCodec{}).Decode(
			encoded[:length],
		); !errors.Is(
			err,
			yacymodel.ErrBadRWIPosting,
		) {
			t.Errorf("truncated to %d bytes: err = %v, want ErrBadRWIPosting", length, err)
		}
	}
}
