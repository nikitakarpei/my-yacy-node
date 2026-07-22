package markdown_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeeds/markdown"
)

func samplePage() pageabsorption.CrawledPage {
	return pageabsorption.CrawledPage{
		CanonicalURL: "http://example.com/a",
		Title:        "Hi",
		Language:     "en",
		CrawledAt:    time.Unix(1_700_000_000, 0),
	}
}

func TestMarkdownFeedFramesTheRenderedMarkdown(t *testing.T) {
	feed := markdown.New()
	publication, err := feed.Frame(samplePage(), []byte("# hi"))
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	messages := publication.Messages()
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	page, err := yacycrawlcontract.UnmarshalPageMarkdownRepresentation(messages[0])
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(page.Markdown) != "# hi" {
		t.Fatalf("markdown = %q", page.Markdown)
	}
}

func TestMarkdownFeedDeclaresRepresentationAndContentFormat(t *testing.T) {
	feed := markdown.New()
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindMarkdown {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if feed.ContentFormat() != contentformatgraph.FormatMarkdown {
		t.Fatalf("content format = %q", feed.ContentFormat())
	}
}
