package urlmeta

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func fullURLMetadata() yacymodel.URLMetadata {
	return yacymodel.URLMetadata{
		Address:          "https://example.org/",
		Referrer:         yacymodel.URLHash("MNOPQRSTUVWX"),
		Title:            "Example, Inc.",
		Author:           "A. Author",
		Tags:             []string{"news", "indexof"},
		Publisher:        "Example Press",
		Location:         yacymodel.Coordinates{Latitude: 52.52, Longitude: 13.405},
		Modified:         yacymodel.NewCalendarDay(2025, time.January, 1),
		Loaded:           yacymodel.NewCalendarDay(2025, time.February, 3),
		FreshUntil:       yacymodel.NewCalendarDay(2026, time.January, 1),
		DocumentType:     yacymodel.DocumentTypeText,
		MediaType:        "text/html",
		Language:         yacymodel.Language("en"),
		ByteSize:         1024,
		WordCount:        12,
		LocalLinks:       3,
		ExternalLinks:    4,
		ImageLinks:       5,
		AudioLinks:       6,
		VideoLinks:       7,
		ApplicationLinks: 8,
		Snippet:          "an example",
		FaviconAddress:   "https://example.org/favicon.ico",
	}
}

func TestStoredURLMetadataCodecRoundTripsEveryField(t *testing.T) {
	codec := storedURLMetadataCodec{}

	want := fullURLMetadata()

	encoded, err := codec.Encode(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := codec.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestStoredURLMetadataCodecRoundTripsAbsentValues(t *testing.T) {
	codec := storedURLMetadataCodec{}

	want := yacymodel.URLMetadata{Address: "https://example.org/"}

	encoded, err := codec.Encode(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := codec.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestStoredURLMetadataCodecRejectsEmptyValue(t *testing.T) {
	codec := storedURLMetadataCodec{}

	if _, err := codec.Decode(nil); !errors.Is(
		err, yacymodel.ErrBadURLMetadata,
	) {
		t.Errorf("err = %v, want ErrBadURLMetadata", err)
	}
}

func TestStoredURLMetadataCodecRejectsUnknownFormat(t *testing.T) {
	codec := storedURLMetadataCodec{}

	if _, err := codec.Decode([]byte{0xff}); !errors.Is(
		err, yacymodel.ErrBadURLMetadata,
	) {
		t.Errorf("err = %v, want ErrBadURLMetadata", err)
	}
}

func TestStoredURLMetadataCodecRejectsTruncatedValue(t *testing.T) {
	codec := storedURLMetadataCodec{}

	encoded, err := codec.Encode(fullURLMetadata())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.Decode(
		encoded[:len(encoded)-1],
	); !errors.Is(err, yacymodel.ErrBadURLMetadata) {
		t.Errorf("err = %v, want ErrBadURLMetadata", err)
	}
}
