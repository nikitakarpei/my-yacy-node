package pagereading_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
)

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/")
}

func page() pagefetch.FetchedPage {
	return pagefetch.FetchedPage{ContentType: "text/html", Body: []byte("body")}
}

func readingOf(
	t *testing.T,
	page pagefetch.FetchedPage,
	pageFormats ...pagereading.PageFormat,
) pagereading.Reading {
	t.Helper()
	reading, err := pagereading.NewPageReader(pageFormats, &recordingPageReadingObserver{}).
		ReadingOfPage(t.Context(), pageURL(t), page)
	if err != nil {
		t.Fatalf("ReadingOfPage: %v", err)
	}
	return reading
}

func TestReadingOfPageReportsNoLinksOnABodyNoFormatKnows(t *testing.T) {
	reading := readingOf(t, page(), unknownFormat{})

	if len(reading.DiscoveredURLs) != 0 {
		t.Fatalf("a body no format knows links nowhere, got %v", reading.DiscoveredURLs)
	}
}

func TestReadingOfPageHonorsTheHeadersOfABodyNoFormatKnows(t *testing.T) {
	fetchedPage := page()
	fetchedPage.RobotsTagValues = []string{"noindex"}

	reading := readingOf(t, fetchedPage, unknownFormat{})

	if !reading.Refusals.RefusesIndexing {
		t.Fatalf("want the header noindex read, got %+v", reading.Refusals)
	}
}

func TestReadingOfPageReportsTheURLsOfTheFirstFormatThatKnowsTheBody(t *testing.T) {
	first := linkingFormat(t, "http://host/first")
	second := linkingFormat(t, "http://host/second")

	reading := readingOf(t, page(), unknownFormat{}, first, second)

	assertURLs(t, reading.DiscoveredURLs, "http://host/first")
}

func TestReadingOfPageKeepsAFormatReadingFailure(t *testing.T) {
	bodyUnreadable := errors.New("body unreadable")
	reader := pagereading.NewPageReader(
		[]pagereading.PageFormat{
			failingFormat{err: bodyUnreadable},
			linkingFormat(t, "http://host/next"),
		},
		&recordingPageReadingObserver{},
	)

	_, err := reader.ReadingOfPage(t.Context(), pageURL(t), page())

	if !errors.Is(err, bodyUnreadable) {
		t.Fatalf("want %v, got %v", bodyUnreadable, err)
	}
}

func TestReadingOfPageStillReportsLinksOnAPageThatRefusesIndexing(t *testing.T) {
	format := linkingFormat(t, "http://host/next")
	format.reading.Refusals = robotsmeta.Refusals{RefusesIndexing: true}

	reading := readingOf(t, page(), format)

	if !reading.Refusals.RefusesIndexing {
		t.Fatalf("want the body noindex kept, got %+v", reading.Refusals)
	}
	assertURLs(t, reading.DiscoveredURLs, "http://host/next")
}

func TestReadingOfPageJoinsTheRefusalsOfTheHeadersAndTheBody(t *testing.T) {
	format := linkingFormat(t, "http://host/next")
	format.reading.Refusals = robotsmeta.Refusals{RefusesIndexing: true}
	fetchedPage := page()
	fetchedPage.RobotsTagValues = []string{"nofollow"}

	reading := readingOf(t, fetchedPage, format)

	if !reading.Refusals.RefusesIndexing || !reading.Refusals.RefusesLinkDiscovery {
		t.Fatalf("want both refusals, got %+v", reading.Refusals)
	}
}

func TestReadingOfPageReportsNoLinksWhenThePageRefusesLinkDiscovery(t *testing.T) {
	bodyRefusal := linkingFormat(t, "http://host/next")
	bodyRefusal.reading.Refusals = robotsmeta.Refusals{RefusesLinkDiscovery: true}
	headerRefusal := page()
	headerRefusal.RobotsTagValues = []string{"nofollow"}
	for name, test := range map[string]struct {
		page   pagefetch.FetchedPage
		format pagereading.PageFormat
	}{
		"body":   {page: page(), format: bodyRefusal},
		"header": {page: headerRefusal, format: linkingFormat(t, "http://host/next")},
	} {
		t.Run(name, func(t *testing.T) {
			observer := &recordingPageReadingObserver{}
			reading, err := pagereading.NewPageReader(
				[]pagereading.PageFormat{test.format}, observer,
			).ReadingOfPage(t.Context(), pageURL(t), test.page)
			if err != nil {
				t.Fatalf("ReadingOfPage: %v", err)
			}

			if len(reading.DiscoveredURLs) != 0 {
				t.Fatalf("nofollow suppresses linked urls, got %v", reading.DiscoveredURLs)
			}
			if observer.linkDiscoveryRefusalsEnforced != 1 {
				t.Fatalf(
					"link discovery refusals enforced = %d, want 1",
					observer.linkDiscoveryRefusalsEnforced,
				)
			}
		})
	}
}

func TestReadingOfPageGivesEveryFormatTheContentTypeOfThePage(t *testing.T) {
	format := &contentTypeRecordingFormat{}
	fetchedPage := page()
	fetchedPage.ContentType = "Text/XML; charset=ISO-8859-1"

	readingOf(t, fetchedPage, format)

	if format.contentType.MediaType() != "text/xml" ||
		format.contentType.Charset() != "ISO-8859-1" ||
		format.contentType.String() != fetchedPage.ContentType {
		t.Fatalf("content type = %+v", format.contentType)
	}
}

func TestReadingOfPageReportsAContentTypeThatCannotBeParsed(t *testing.T) {
	observer := &recordingPageReadingObserver{}
	format := &contentTypeRecordingFormat{}
	fetchedPage := page()
	fetchedPage.ContentType = "Text/HTML; charset"

	_, err := pagereading.NewPageReader([]pagereading.PageFormat{format}, observer).
		ReadingOfPage(t.Context(), pageURL(t), fetchedPage)
	if err != nil {
		t.Fatalf("ReadingOfPage: %v", err)
	}

	if len(observer.unparsedContentTypes) != 1 ||
		observer.unparsedContentTypes[0] != fetchedPage.ContentType {
		t.Fatalf("unparsed content types = %v", observer.unparsedContentTypes)
	}
	if format.contentType.MediaType() != "text/html" {
		t.Fatalf("want the leading segment as media type, got %q", format.contentType.MediaType())
	}
}

func TestEveryObserverHearsWhatThePageReaderReports(t *testing.T) {
	first, second := &recordingPageReadingObserver{}, &recordingPageReadingObserver{}
	fetchedPage := page()
	fetchedPage.ContentType = "text/html; charset"
	fetchedPage.RobotsTagValues = []string{"nofollow"}

	_, err := pagereading.NewPageReader(
		[]pagereading.PageFormat{linkingFormat(t, "http://host/next")},
		pagereading.PageReadingObservers{first, second},
	).ReadingOfPage(t.Context(), pageURL(t), fetchedPage)
	if err != nil {
		t.Fatalf("ReadingOfPage: %v", err)
	}

	for _, observer := range []*recordingPageReadingObserver{first, second} {
		if len(observer.unparsedContentTypes) != 1 || observer.linkDiscoveryRefusalsEnforced != 1 {
			t.Fatalf("observer heard %+v, want one of each", *observer)
		}
	}
}

func assertURLs(t *testing.T, got []canonicalurl.CanonicalURL, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("urls = %v, want %v", got, want)
	}
	for i, wanted := range want {
		if got[i].String() != wanted {
			t.Fatalf("urls = %v, want %v", got, want)
		}
	}
}

type unknownFormat struct{}

func (unknownFormat) ReadingOfBody(
	context.Context, canonicalurl.CanonicalURL, pagereading.ContentType, []byte,
) (pagereading.BodyReading, bool, error) {
	return pagereading.BodyReading{}, false, nil
}

type knownFormat struct {
	reading pagereading.BodyReading
}

func linkingFormat(t *testing.T, linkedURL string) *knownFormat {
	t.Helper()
	return &knownFormat{reading: pagereading.BodyReading{
		LinkedURLs: []canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, linkedURL)},
	}}
}

func (format *knownFormat) ReadingOfBody(
	context.Context, canonicalurl.CanonicalURL, pagereading.ContentType, []byte,
) (pagereading.BodyReading, bool, error) {
	return format.reading, true, nil
}

type failingFormat struct{ err error }

func (format failingFormat) ReadingOfBody(
	context.Context, canonicalurl.CanonicalURL, pagereading.ContentType, []byte,
) (pagereading.BodyReading, bool, error) {
	return pagereading.BodyReading{}, true, format.err
}

type contentTypeRecordingFormat struct {
	contentType pagereading.ContentType
}

func (format *contentTypeRecordingFormat) ReadingOfBody(
	_ context.Context, _ canonicalurl.CanonicalURL, contentType pagereading.ContentType, _ []byte,
) (pagereading.BodyReading, bool, error) {
	format.contentType = contentType
	return pagereading.BodyReading{}, true, nil
}

type recordingPageReadingObserver struct {
	unparsedContentTypes          []string
	linkDiscoveryRefusalsEnforced int
}

func (o *recordingPageReadingObserver) MediaTypeUnparsed(
	_ context.Context, contentType string, _ error,
) {
	o.unparsedContentTypes = append(o.unparsedContentTypes, contentType)
}

func (o *recordingPageReadingObserver) LinkDiscoveryRefusalEnforced(
	context.Context, canonicalurl.CanonicalURL,
) {
	o.linkDiscoveryRefusalsEnforced++
}
