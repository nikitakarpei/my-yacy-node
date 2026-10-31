package yacyproto_test

import (
	"math"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const yacyURLMetadataRow = "{hash=MNOPQRSTUVWX,url=b|aHR0cHM6Ly9leGFtcGxlLm9yZy8," +
	"descr=b|RXhhbXBsZQ,author=b|,tags=b|,publisher=b|,lat=0,lon=0," +
	"mod=20250101,load=20250203,fresh=20260101,referrer=,size=1024,wc=12," +
	"dt=t,flags=AAAAAA,lang=en,llocal=3,lother=4,limage=0,laudio=0," +
	"lvideo=0,lapp=0}"

func samplePostingsRoot(t *testing.T) yacymodel.PostingsRoot {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("partitions: %v", err)
	}
	crawled, err := yacymodel.ProvenPagePostingsFrom(
		[]yacymodel.RWIPosting{{
			WordHash: yacymodel.WordHash("sample"),
			URLHash:  sampleURLHash(t, "sample"),
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

func fullURLMetadata(t *testing.T) yacymodel.URLMetadata {
	t.Helper()

	return yacymodel.URLMetadata{
		Hash:             urlhashtest.URLHashOf(t, "https://example.org/"),
		Address:          "https://example.org/",
		Referrer:         yacymodel.Some(sampleURLHash(t, "referrer")),
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
		Language:         yacymodel.Some(englishLanguage(t)),
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
}

func metadataFromRow(t *testing.T, row string) yacymodel.URLMetadata {
	t.Helper()

	form := url.Values{
		yacyproto.FieldURLCount: {"1"},
		"url0":                  {row},
	}
	req, err := yacyproto.ParseTransferURLRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseTransferURLRequest: %v", err)
	}
	if len(req.URLs) != 1 {
		t.Fatalf("URLs = %d, want 1 for row %q", len(req.URLs), row)
	}

	return req.URLs[0]
}

func TestTransferURLRequestCarriesEveryMetadataColumn(t *testing.T) {
	t.Parallel()

	want := fullURLMetadata(t)

	got := metadataFromRow(t, sampleURLMetadataWireForm(t, want))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestTransferURLRequestReadsARowAsRealYaCyWritesIt(t *testing.T) {
	t.Parallel()

	want := yacymodel.URLMetadata{
		Hash:          mustParseURLHash(t, "MNOPQRSTUVWX"),
		Address:       "https://example.org/",
		Title:         "Example",
		Modified:      yacymodel.Some(yacymodel.NewCalendarDay(2025, time.January, 1)),
		Loaded:        yacymodel.Some(yacymodel.NewCalendarDay(2025, time.February, 3)),
		FreshUntil:    yacymodel.Some(yacymodel.NewCalendarDay(2026, time.January, 1)),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      yacymodel.Some(englishLanguage(t)),
		ByteSize:      yacymodel.Some[uint32](1024),
		WordCount:     12,
		LocalLinks:    3,
		ExternalLinks: 4,
	}

	if got := metadataFromRow(t, yacyURLMetadataRow); !reflect.DeepEqual(got, want) {
		t.Errorf("metadata =\n %+v\nwant\n %+v", got, want)
	}
}

func TestTransferURLRequestCarriesCommasInText(t *testing.T) {
	t.Parallel()

	const address = "http://example.com/article?ids=1,2,3"

	want := yacymodel.URLMetadata{
		Hash:    urlhashtest.URLHashOf(t, address),
		Address: address,
		Title:   "Fourth of July fireworks, 1986 - Example",
	}

	got := metadataFromRow(t, sampleURLMetadataWireForm(t, want))
	if got.Address != want.Address || got.Title != want.Title {
		t.Errorf("comma-bearing text did not survive: %+v", got)
	}
}

func TestTransferURLRequestWritesTheHashTheMetadataNames(t *testing.T) {
	t.Parallel()

	metadata := fullURLMetadata(t)
	metadata.Hash = mustParseURLHash(t, "MNOPQRSTUVWX")

	row := sampleURLMetadataWireForm(t, metadata)

	if !strings.Contains(row, "hash=MNOPQRSTUVWX") {
		t.Errorf("row does not carry the hash the metadata names: %s", row)
	}
}

func TestTransferURLRequestWritesTheFlagsColumnFromTheMetadata(t *testing.T) {
	t.Parallel()

	row := sampleURLMetadataWireForm(t, fullURLMetadata(t))

	if strings.Contains(row, "flags=AAAAAA") {
		t.Errorf("metadata with tags, location and media should set flags: %s", row)
	}
}

func TestTransferURLRequestReadsAnUnknownSizeAsAbsent(t *testing.T) {
	t.Parallel()

	for _, row := range []string{
		strings.Replace(yacyURLMetadataRow, "size=1024", "size=-1", 1),
		strings.Replace(yacyURLMetadataRow, "size=1024,", "", 1),
	} {
		if size := metadataFromRow(t, row).ByteSize; size.Present() {
			t.Errorf("byte size = %v, want absent for row %q", size, row)
		}
	}
}

func TestTransferURLRequestReadsANegativeCountAsZero(t *testing.T) {
	t.Parallel()

	row := strings.Replace(yacyURLMetadataRow, "wc=12", "wc=-3", 1)

	if count := metadataFromRow(t, row).WordCount; count != 0 {
		t.Errorf("word count = %d, want zero", count)
	}
}

func TestTransferURLRequestHoldsACountWiderThanFourBytesAtItsCeiling(t *testing.T) {
	t.Parallel()

	for written, want := range map[string]uint32{
		"4294967295": math.MaxUint32,
		"4294967296": math.MaxUint32,
		"5000000000": math.MaxUint32,
	} {
		row := strings.Replace(yacyURLMetadataRow, "wc=12", "wc="+written, 1)
		row = strings.Replace(row, "size=1024", "size="+written, 1)
		metadata := metadataFromRow(t, row)
		if metadata.WordCount != want {
			t.Errorf("word count of %s = %d, want %d", written, metadata.WordCount, want)
		}
		if size, _ := metadata.ByteSize.Get(); size != want {
			t.Errorf("byte size of %s = %d, want %d", written, size, want)
		}
	}
}

func TestTransferURLRequestCarriesCountsAtTheirCeiling(t *testing.T) {
	t.Parallel()

	written := fullURLMetadata(t)
	written.ByteSize = yacymodel.Some[uint32](math.MaxUint32)
	written.WordCount = math.MaxUint32
	written.LocalLinks = math.MaxUint32
	written.ExternalLinks = math.MaxUint32
	written.ImageLinks = math.MaxUint32
	written.AudioLinks = math.MaxUint32
	written.VideoLinks = math.MaxUint32
	written.ApplicationLinks = math.MaxUint32

	if got := metadataFromRow(
		t,
		sampleURLMetadataWireForm(t, written),
	); !reflect.DeepEqual(
		got,
		written,
	) {
		t.Errorf("round trip =\n %+v\nwant\n %+v", got, written)
	}
}

func TestTransferURLRequestWritesAnAbsentSizeAsYaCyDoes(t *testing.T) {
	t.Parallel()

	metadata := fullURLMetadata(t)
	metadata.ByteSize = yacymodel.None[uint32]()

	if row := sampleURLMetadataWireForm(t, metadata); !strings.Contains(row, "size=-1,") {
		t.Errorf("row does not carry the unknown size as -1: %s", row)
	}
}

func TestTransferURLRequestWritesNoPostingsRootForMetadataWithoutOne(t *testing.T) {
	t.Parallel()

	metadata := fullURLMetadata(t)
	metadata.PostingsRoot = yacymodel.None[yacymodel.PostingsRoot]()

	row := sampleURLMetadataWireForm(t, metadata)
	if strings.Contains(row, "postingsroot=") {
		t.Errorf("row carries a postings root the metadata does not have: %s", row)
	}
	if got := metadataFromRow(t, row); !reflect.DeepEqual(got, metadata) {
		t.Errorf("round trip =\n %+v\nwant\n %+v", got, metadata)
	}
}

func TestTransferURLRequestReadsABadPostingsRootAsAbsent(t *testing.T) {
	t.Parallel()

	want := metadataFromRow(t, yacyURLMetadataRow)
	shortRoot := yacymodel.Encode(samplePostingsRoot(t).Bytes()[:10])
	for _, value := range []string{"not*base64", shortRoot} {
		row := strings.Replace(yacyURLMetadataRow, "}", ",postingsroot="+value+"}", 1)
		if got := metadataFromRow(t, row); !reflect.DeepEqual(got, want) {
			t.Errorf("row with postings root %q =\n %+v\nwant\n %+v", value, got, want)
		}
	}
}
