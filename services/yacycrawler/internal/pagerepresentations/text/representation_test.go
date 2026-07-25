package text_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerepresentations/text"
)

func samplePage() pagepublication.Page {
	return pagepublication.Page{
		CanonicalURL: "http://example.com/a",
		Title:        "Hi",
		Language:     "en",
		CrawledAt:    time.Unix(1_700_000_000, 0),
	}
}

func TestRepresentationFramesTheRenderedText(t *testing.T) {
	representation := text.New()
	publication, err := representation.Frame(samplePage(), []byte("the quick brown fox"))
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	messages := publication
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	page, err := yacycrawlcontract.UnmarshalPageTextRepresentation(messages[0])
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if page.Title != "Hi" {
		t.Fatalf("title = %q", page.Title)
	}
	if string(page.Text) != "the quick brown fox" {
		t.Fatalf("text = %q", page.Text)
	}
}

func TestRepresentationDeclaresKindAndContentFormat(t *testing.T) {
	representation := text.New()
	if representation.Kind() != yacycrawlcontract.PageRepresentationKindText {
		t.Fatalf("representation = %q", representation.Kind())
	}
	if representation.ContentFormat() != contentformatgraph.FormatReadableText {
		t.Fatalf("content format = %q", representation.ContentFormat())
	}
}
