package urlmetarecord_test

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta/urlmetarecord"
)

func samplePostingsRoot(t *testing.T) yacymodel.PostingsRoot {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("partitions: %v", err)
	}
	crawled, err := yacymodel.ProvenPagePostingsFrom(
		[]yacymodel.RWIPosting{{
			WordHash: yacymodel.WordHash("sample"),
			URLHash:  urlhashtest.URLHashOf(t, "https://example.org/sample"),
		}},
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

func mustLanguage(t *testing.T, raw string) yacymodel.Optional[yacymodel.Language] {
	t.Helper()

	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		t.Fatalf("parse language %q: %v", raw, err)
	}

	return yacymodel.Some(language)
}

func fullRecord(t *testing.T) urlmetarecord.Record {
	t.Helper()

	return urlmetarecord.Record{
		Metadata: yacymodel.URLMetadata{
			Address: "https://example.org/",
			Referrer: yacymodel.Some(
				urlhashtest.URLHashOf(t, "https://example.org/referrer"),
			),
			Title:     "Example, Inc.",
			Author:    "A. Author",
			Tags:      []string{"news", "indexof"},
			Publisher: "Example Press",
			Location: yacymodel.Some(
				yacymodel.Coordinates{Latitude: 52.52, Longitude: 13.405},
			),
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
		},
		Source: pageversion.SourceThisNode(),
	}
}

func encoded(t *testing.T, record urlmetarecord.Record) []byte {
	t.Helper()

	raw, err := urlmetarecord.Codec{}.Encode(record)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	return raw
}

func decoded(t *testing.T, raw []byte) urlmetarecord.Record {
	t.Helper()

	record, err := urlmetarecord.Codec{}.Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	return record
}

func TestStoredURLMetadataKeepsEveryField(t *testing.T) {
	stored := fullRecord(t)

	if read := decoded(t, encoded(t, stored)); !reflect.DeepEqual(read, stored) {
		t.Errorf("read back\n got  %+v\n want %+v", read, stored)
	}
}

func TestStoredURLMetadataKeepsCountsAtTheirCeiling(t *testing.T) {
	stored := fullRecord(t)
	stored.Metadata.ByteSize = yacymodel.Some[uint32](math.MaxUint32)
	stored.Metadata.WordCount = math.MaxUint32
	stored.Metadata.LocalLinks = math.MaxUint32
	stored.Metadata.ExternalLinks = math.MaxUint32
	stored.Metadata.ImageLinks = math.MaxUint32
	stored.Metadata.AudioLinks = math.MaxUint32
	stored.Metadata.VideoLinks = math.MaxUint32
	stored.Metadata.ApplicationLinks = math.MaxUint32

	if read := decoded(t, encoded(t, stored)); !reflect.DeepEqual(read, stored) {
		t.Errorf("read back\n got  %+v\n want %+v", read, stored)
	}
}

func TestStoredURLMetadataKeepsAbsentValues(t *testing.T) {
	stored := urlmetarecord.Record{
		Metadata: yacymodel.URLMetadata{Address: "https://example.org/"},
		Source:   pageversion.SourceThisNode(),
	}

	if read := decoded(t, encoded(t, stored)); !reflect.DeepEqual(read, stored) {
		t.Errorf("read back\n got  %+v\n want %+v", read, stored)
	}
}

func withoutPostingsRoot(record urlmetarecord.Record) urlmetarecord.Record {
	record.Metadata.PostingsRoot = yacymodel.None[yacymodel.PostingsRoot]()

	return record
}

func TestARecordStoredBeforeThePostingsRootReadsWithoutOne(t *testing.T) {
	record := fullRecord(t)
	withRoot := encoded(t, record)
	withoutRoot := encoded(t, withoutPostingsRoot(record))
	sharedLength := 0
	for sharedLength < len(withoutRoot) && withRoot[sharedLength] == withoutRoot[sharedLength] {
		sharedLength++
	}

	read, want := decoded(t, withRoot[:sharedLength]), withoutPostingsRoot(record)
	if !reflect.DeepEqual(read, want) {
		t.Errorf("read back\n got  %+v\n want %+v", read, want)
	}
}

func TestACorruptStoredPostingsRootRejectsTheRecord(t *testing.T) {
	record := fullRecord(t)
	root, _ := record.Metadata.PostingsRoot.Get()
	raw := encoded(t, record)
	raw[len(raw)-len(root.Bytes())] = 0

	if _, err := (urlmetarecord.Codec{}).Decode(raw); !errors.Is(err, yacymodel.ErrBadURLMetadata) {
		t.Errorf("Decode error = %v, want %v", err, yacymodel.ErrBadURLMetadata)
	}
}
