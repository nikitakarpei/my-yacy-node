package judgedqueries_test

import (
	"strings"
	"testing"
)

func TestTheExtractedPageOfAnHTMLPageCountsItsLinksAndHoldsItsTitleAndText(t *testing.T) {
	t.Parallel()

	extractedPage := pageExtractionOfEveryFormat(t).extractedPageOf(t.Context(), storedPage{
		address:     "https://example.org/weather",
		contentType: "text/html; charset=utf-8",
		body: []byte(`<html><head><title>Weather in Berlin</title></head><body>
			<p>Rain is falling over the whole city today.</p>
			<a href="/tomorrow">Tomorrow</a>
			<a href="/yesterday">Yesterday</a>
			<a href="https://other.example/radar">Radar</a>
		</body></html>`),
	})

	if extractedPage.title != "Weather in Berlin" {
		t.Fatalf("the extractedPage page holds the title %q, want %q",
			extractedPage.title, "Weather in Berlin")
	}
	if !strings.Contains(extractedPage.text, "Rain is falling") {
		t.Fatalf("the extractedPage page holds the text %q, want the text of the page",
			extractedPage.text)
	}
	if extractedPage.linkCounts.LocalLinks != 2 || extractedPage.linkCounts.ExternalLinks != 1 {
		t.Fatalf("the extractedPage page holds the link counts %+v, want 2 local and 1 external",
			extractedPage.linkCounts)
	}
}

func TestTheExtractedPageOfAPageOfNoKnownFormatHoldsNothing(t *testing.T) {
	t.Parallel()

	extractedPage := pageExtractionOfEveryFormat(t).extractedPageOf(t.Context(), storedPage{
		address:     "https://example.org/opaque",
		contentType: "application/octet-stream",
		body:        []byte{0x00, 0x01, 0x02},
	})

	if extractedPage.title != "" || extractedPage.text != "" {
		t.Fatalf("the extractedPage page reads %+v, want nothing", extractedPage)
	}
}
