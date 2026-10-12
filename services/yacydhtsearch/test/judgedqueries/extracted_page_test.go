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

	if extractedPage.Title != "Weather in Berlin" {
		t.Fatalf("the extractedPage page holds the title %q, want %q",
			extractedPage.Title, "Weather in Berlin")
	}
	if !strings.Contains(extractedPage.Text, "Rain is falling") {
		t.Fatalf("the extractedPage page holds the text %q, want the text of the page",
			extractedPage.Text)
	}
	if extractedPage.LinkCounts.LocalLinks != 2 || extractedPage.LinkCounts.ExternalLinks != 1 {
		t.Fatalf("the extractedPage page holds the link counts %+v, want 2 local and 1 external",
			extractedPage.LinkCounts)
	}
}

func TestTheExtractedPageOfAPageOfNoKnownFormatHoldsNothing(t *testing.T) {
	t.Parallel()

	extractedPage := pageExtractionOfEveryFormat(t).extractedPageOf(t.Context(), storedPage{
		address:     "https://example.org/opaque",
		contentType: "application/octet-stream",
		body:        []byte{0x00, 0x01, 0x02},
	})

	if extractedPage.Title != "" || extractedPage.Text != "" {
		t.Fatalf("the extractedPage page reads %+v, want nothing", extractedPage)
	}
}
