package yacyproto

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// yacyURLMetadataRow is shaped as YaCy's own corePropList writes it, in its
// emission order and with only the columns YaCy emits for a plain text page.
const yacyURLMetadataRow = "{hash=MNOPQRSTUVWX,url=b|aHR0cHM6Ly9leGFtcGxlLm9yZy8," +
	"descr=b|RXhhbXBsZQ,author=b|,tags=b|,publisher=b|,lat=0,lon=0," +
	"mod=20250101,load=20250203,fresh=20260101,referrer=,size=1024,wc=12," +
	"dt=t,flags=AAAAAA,lang=en,llocal=3,lother=4,limage=0,laudio=0," +
	"lvideo=0,lapp=0}"

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

func TestURLMetadataWireCodecRoundTripsEveryColumn(t *testing.T) {
	codec := urlMetadataWireCodec{}
	want := fullURLMetadata()

	got, err := codec.decode(context.Background(), codec.encode(want))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestURLMetadataWireCodecDecodesYaCyRow(t *testing.T) {
	got, err := urlMetadataWireCodec{}.decode(context.Background(), yacyURLMetadataRow)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	want := yacymodel.URLMetadata{
		Address:       "https://example.org/",
		Title:         "Example",
		Modified:      yacymodel.NewCalendarDay(2025, time.January, 1),
		Loaded:        yacymodel.NewCalendarDay(2025, time.February, 3),
		FreshUntil:    yacymodel.NewCalendarDay(2026, time.January, 1),
		DocumentType:  yacymodel.DocumentTypeText,
		Language:      yacymodel.Language("en"),
		ByteSize:      1024,
		WordCount:     12,
		LocalLinks:    3,
		ExternalLinks: 4,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decode =\n %+v\nwant\n %+v", got, want)
	}
}

func TestURLMetadataWireCodecCarriesCommasInText(t *testing.T) {
	codec := urlMetadataWireCodec{}
	want := yacymodel.URLMetadata{
		Address: "http://example.com/article?ids=1,2,3",
		Title:   "Fourth of July fireworks, 1986 - Example",
	}

	got, err := codec.decode(context.Background(), codec.encode(want))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Address != want.Address || got.Title != want.Title {
		t.Errorf("comma-bearing text did not survive: %+v", got)
	}
}

func TestURLMetadataWireCodecEncodesHashFromAddress(t *testing.T) {
	row := urlMetadataWireCodec{}.encode(fullURLMetadata())

	hash, err := yacymodel.HashURL("https://example.org/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(row, urlMetadataColHash+"="+hash.String()) {
		t.Errorf("row does not carry the address hash: %s", row)
	}
}

func TestURLMetadataWireCodecEncodesDerivedFlags(t *testing.T) {
	row := urlMetadataWireCodec{}.encode(fullURLMetadata())

	if strings.Contains(row, urlMetadataColFlags+"=AAAAAA") {
		t.Errorf("metadata with tags, location and media should set flags: %s", row)
	}
}

func TestURLMetadataWireCodecRejectsMalformedRows(t *testing.T) {
	codec := urlMetadataWireCodec{}
	for _, c := range []struct{ name, row string }{
		{"no property form", "hash=MNOPQRSTUVWX"},
		{"empty", ""},
		{"undecodable text", "{url=z|@@@}"},
		{"bad referrer", "{url=b|aHR0cDovL2Ev,referrer=short}"},
		{"out of range location", "{url=b|aHR0cDovL2Ev,lat=91,lon=0}"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := codec.decode(context.Background(), c.row); !errors.Is(
				err, yacymodel.ErrBadURLMetadata,
			) {
				t.Errorf("err = %v, want ErrBadURLMetadata", err)
			}
		})
	}
}
