package urlmeta_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
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

func samplePostingsRoot(t *testing.T) yacymodel.PostingsRoot {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("partitions: %v", err)
	}
	crawled, err := yacymodel.ProvenPagePostingsFrom(
		[]yacymodel.RWIPosting{{WordHash: yacymodel.WordHash("sample")}},
		yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		t.Fatalf("crawled page postings: %v", err)
	}
	root, rooted := crawled.Root().Get()
	if !rooted {
		t.Fatal("the crawled page postings have no root")
	}

	return root
}

func fullURLMetadata(t *testing.T) yacymodel.URLMetadata {
	t.Helper()

	metadata := yacymodel.URLMetadata{
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
		ByteSize:         yacymodel.Some[uint32](1024),
		WordCount:        12,
		LocalLinks:       3,
		ExternalLinks:    4,
		ImageLinks:       5,
		AudioLinks:       6,
		VideoLinks:       7,
		ApplicationLinks: 8,
		Snippet:          "an example",
		FaviconAddress:   "https://example.org/favicon.ico",
		PostingsRoot:     yacymodel.Some(samplePostingsRoot(t)),
	}
	metadata.Hash = urlhashtest.URLHashOf(t, metadata.Address)

	return metadata
}

func writtenThenRead(t *testing.T, metadata yacymodel.URLMetadata) yacymodel.URLMetadata {
	t.Helper()

	v, module := openObservedModule(t)
	writeAll(t, v, module, metadata)

	hash := metadata.Hash
	rows := metadataPerHash(t, v, module.Directory, []yacymodel.URLHash{hash})
	if len(rows) != 1 {
		t.Fatalf("rows = %v, want the one written row", rows)
	}

	return rows[hash]
}

func TestStoredURLMetadataKeepsEveryField(t *testing.T) {
	stored := fullURLMetadata(t)

	if read := writtenThenRead(t, stored); !reflect.DeepEqual(read, stored) {
		t.Errorf("read back\n got  %+v\n want %+v", read, stored)
	}
}

func TestStoredURLMetadataKeepsCountsAtTheirCeiling(t *testing.T) {
	stored := fullURLMetadata(t)
	stored.ByteSize = yacymodel.Some[uint32](math.MaxUint32)
	stored.WordCount = math.MaxUint32
	stored.LocalLinks = math.MaxUint32
	stored.ExternalLinks = math.MaxUint32
	stored.ImageLinks = math.MaxUint32
	stored.AudioLinks = math.MaxUint32
	stored.VideoLinks = math.MaxUint32
	stored.ApplicationLinks = math.MaxUint32

	if read := writtenThenRead(t, stored); !reflect.DeepEqual(read, stored) {
		t.Errorf("read back\n got  %+v\n want %+v", read, stored)
	}
}

func TestStoredURLMetadataKeepsAbsentValues(t *testing.T) {
	const address = "https://example.org/"

	stored := yacymodel.URLMetadata{Hash: urlhashtest.URLHashOf(t, address), Address: address}

	if read := writtenThenRead(t, stored); !reflect.DeepEqual(read, stored) {
		t.Errorf("read back\n got  %+v\n want %+v", read, stored)
	}
}
