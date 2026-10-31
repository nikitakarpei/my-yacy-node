package urlmetarecord_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

func rootedRecord(t *testing.T) urlmetarecord.Record {
	t.Helper()

	return urlmetarecord.Record{
		Metadata: yacymodel.URLMetadata{
			Address:      "https://example.org/",
			Title:        "Example",
			Loaded:       yacymodel.Some(yacymodel.NewCalendarDay(2025, time.February, 3)),
			WordCount:    12,
			Snippet:      "an example",
			PostingsRoot: yacymodel.Some(samplePostingsRoot(t)),
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

func withoutPostingsRoot(record urlmetarecord.Record) urlmetarecord.Record {
	record.Metadata.PostingsRoot = yacymodel.None[yacymodel.PostingsRoot]()

	return record
}

func TestARecordReadsBackWithItsPostingsRoot(t *testing.T) {
	record := rootedRecord(t)

	read, err := urlmetarecord.Codec{}.Decode(encoded(t, record))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(read, record) {
		t.Errorf("read back\n got  %+v\n want %+v", read, record)
	}
}

func TestARecordStoredBeforeThePostingsRootReadsWithoutOne(t *testing.T) {
	record := rootedRecord(t)
	withRoot := encoded(t, record)
	withoutRoot := encoded(t, withoutPostingsRoot(record))
	sharedLength := 0
	for sharedLength < len(withoutRoot) && withRoot[sharedLength] == withoutRoot[sharedLength] {
		sharedLength++
	}

	read, err := urlmetarecord.Codec{}.Decode(withRoot[:sharedLength])
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if want := withoutPostingsRoot(record); !reflect.DeepEqual(read, want) {
		t.Errorf("read back\n got  %+v\n want %+v", read, want)
	}
}

func TestACorruptStoredPostingsRootRejectsTheRecord(t *testing.T) {
	record := rootedRecord(t)
	raw := encoded(t, record)
	raw[len(raw)-len(samplePostingsRoot(t).Bytes())] = 0

	if _, err := (urlmetarecord.Codec{}).Decode(raw); !errors.Is(err, yacymodel.ErrBadURLMetadata) {
		t.Errorf("Decode error = %v, want %v", err, yacymodel.ErrBadURLMetadata)
	}
}
