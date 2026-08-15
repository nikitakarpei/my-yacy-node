package urlmeta_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func mustURLHash(t *testing.T, raw string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.ParseURLHash(raw)
	if err != nil {
		t.Fatalf("parse url hash %q: %v", raw, err)
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

func fullURLMetadata(t *testing.T) yacymodel.URLMetadata {
	t.Helper()

	return yacymodel.URLMetadata{
		Address:          "https://example.org/",
		Referrer:         yacymodel.Some(mustURLHash(t, "MNOPQRSTUVWX")),
		Title:            "Example, Inc.",
		Author:           "A. Author",
		Tags:             []string{"news", "indexof"},
		Publisher:        "Example Press",
		Location:         yacymodel.Some(yacymodel.Coordinates{Latitude: 52.52, Longitude: 13.405}),
		Modified:         yacymodel.Some(yacymodel.NewCalendarDay(2025, time.January, 1)),
		Loaded:           yacymodel.Some(yacymodel.NewCalendarDay(2025, time.February, 3)),
		FreshUntil:       yacymodel.Some(yacymodel.NewCalendarDay(2026, time.January, 1)),
		DocumentType:     yacymodel.DocumentTypeText,
		MediaType:        "text/html",
		Language:         mustLanguage(t, "en"),
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

func receivedThenRead(t *testing.T, metadata yacymodel.URLMetadata) yacymodel.URLMetadata {
	t.Helper()

	ctx := context.Background()
	module := openModule(t, 0)
	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{metadata}); err != nil {
		t.Fatalf("Receive: %v", err)
	}

	rows, err := module.Directory.MetadataByHash(
		ctx,
		[]yacymodel.URLHash{metadataHash(t, metadata)},
	)
	if err != nil {
		t.Fatalf("MetadataByHash: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %v, want the one received row", rows)
	}

	return rows[0]
}

func TestStoredURLMetadataKeepsEveryField(t *testing.T) {
	stored := fullURLMetadata(t)

	if read := receivedThenRead(t, stored); !reflect.DeepEqual(read, stored) {
		t.Errorf("read back\n got  %+v\n want %+v", read, stored)
	}
}

func TestStoredURLMetadataKeepsAbsentValues(t *testing.T) {
	stored := yacymodel.URLMetadata{Address: "https://example.org/"}

	if read := receivedThenRead(t, stored); !reflect.DeepEqual(read, stored) {
		t.Errorf("read back\n got  %+v\n want %+v", read, stored)
	}
}
