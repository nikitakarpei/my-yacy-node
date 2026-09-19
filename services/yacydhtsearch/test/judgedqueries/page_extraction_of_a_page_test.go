package judgedqueries_test

import (
	"strings"
	"testing"
)

func TestTheExtractedPageOfAnHTMLPageCountsItsLinksAndHoldsItsTitleAndText(t *testing.T) {
	t.Parallel()

	extracted := pageExtractionOfTheFormats(t).extractedPageOf(t.Context(), storedPage{
		address:     "https://example.org/weather",
		contentType: "text/html; charset=utf-8",
		body: []byte(`<html><head><title>Weather in Berlin</title></head><body>
			<p>Rain is falling over the whole city today.</p>
			<a href="/tomorrow">Tomorrow</a>
			<a href="/yesterday">Yesterday</a>
			<a href="https://other.example/radar">Radar</a>
		</body></html>`),
	})

	if extracted.title != "Weather in Berlin" {
		t.Fatalf("the extracted page holds the title %q, want %q",
			extracted.title, "Weather in Berlin")
	}
	if !strings.Contains(extracted.text, "Rain is falling") {
		t.Fatalf("the extracted page holds the text %q, want the text of the page",
			extracted.text)
	}
	if extracted.linkCounts.LocalLinks != 2 || extracted.linkCounts.ExternalLinks != 1 {
		t.Fatalf("the extracted page holds the link counts %+v, want 2 local and 1 external",
			extracted.linkCounts)
	}
}

func TestTheExtractedPageOfAPageOfNoKnownFormatHoldsNothing(t *testing.T) {
	t.Parallel()

	extracted := pageExtractionOfTheFormats(t).extractedPageOf(t.Context(), storedPage{
		address:     "https://example.org/opaque",
		contentType: "application/octet-stream",
		body:        []byte{0x00, 0x01, 0x02},
	})

	if extracted.title != "" || extracted.text != "" {
		t.Fatalf("the extracted page reads %+v, want nothing", extracted)
	}
}
