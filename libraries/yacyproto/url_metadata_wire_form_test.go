package yacyproto_test

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const yacyURLMetadataRow = "{hash=MNOPQRSTUVWX,url=b|aHR0cHM6Ly9leGFtcGxlLm9yZy8," +
	"descr=b|RXhhbXBsZQ,author=b|,tags=b|,publisher=b|,lat=0,lon=0," +
	"mod=20250101,load=20250203,fresh=20260101,referrer=,size=1024,wc=12," +
	"dt=t,flags=AAAAAA,lang=en,llocal=3,lother=4,limage=0,laudio=0," +
	"lvideo=0,lapp=0}"

func fullURLMetadata(t *testing.T) yacymodel.URLMetadata {
	t.Helper()

	return yacymodel.URLMetadata{
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
		Address:       "https://example.org/",
		Title:         "Example",
		Modified:      yacymodel.Some(yacymodel.NewCalendarDay(2025, time.January, 1)),
		Loaded:        yacymodel.Some(yacymodel.NewCalendarDay(2025, time.February, 3)),
		FreshUntil:    yacymodel.Some(yacymodel.NewCalendarDay(2026, time.January, 1)),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      mustLanguage(t, "en"),
		ByteSize:      1024,
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

	want := yacymodel.URLMetadata{
		Address: "http://example.com/article?ids=1,2,3",
		Title:   "Fourth of July fireworks, 1986 - Example",
	}

	got := metadataFromRow(t, sampleURLMetadataWireForm(t, want))
	if got.Address != want.Address || got.Title != want.Title {
		t.Errorf("comma-bearing text did not survive: %+v", got)
	}
}

func TestTransferURLRequestWritesTheAddressHashColumn(t *testing.T) {
	t.Parallel()

	row := sampleURLMetadataWireForm(t, fullURLMetadata(t))

	address, err := url.Parse("https://example.org/")
	if err != nil {
		t.Fatal(err)
	}
	hash := yacymodel.URLNormalformOf(address).Hash()
	if !strings.Contains(row, "hash="+hash.String()) {
		t.Errorf("row does not carry the address hash: %s", row)
	}
}

func TestTransferURLRequestWritesTheFlagsColumnFromTheMetadata(t *testing.T) {
	t.Parallel()

	row := sampleURLMetadataWireForm(t, fullURLMetadata(t))

	if strings.Contains(row, "flags=AAAAAA") {
		t.Errorf("metadata with tags, location and media should set flags: %s", row)
	}
}
