package text_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeeds/text"
)

func samplePage() pageabsorption.CrawledPage {
	return pageabsorption.CrawledPage{
		CanonicalURL: "http://example.com/a",
		Title:        "Hi",
		Language:     "en",
		CrawledAt:    time.Unix(1_700_000_000, 0),
	}
}

func TestTextFeedFramesTheRenderedText(t *testing.T) {
	feed := text.New()
	publication, err := feed.Frame(samplePage(), []byte("the quick brown fox"))
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	messages := publication.Messages()
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

func TestTextFeedDeclaresRepresentationAndContentFormat(t *testing.T) {
	feed := text.New()
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindText {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if feed.ContentFormat() != contentformatgraph.FormatReadableText {
		t.Fatalf("content format = %q", feed.ContentFormat())
	}
}
