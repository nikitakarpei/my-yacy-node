package markdown_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerepresentations/markdown"
)

func samplePage(t *testing.T) pagepublication.Page {
	return pagepublication.Page{
		CanonicalURL: canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
		Title:        "Hi",
		Language:     "en",
		CrawledAt:    time.Unix(1_700_000_000, 0),
	}
}

func TestRepresentationFramesTheRenderedMarkdown(t *testing.T) {
	representation := markdown.New()
	publication, err := representation.Frame(samplePage(t), []byte("# hi"))
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	messages := publication
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

func TestRepresentationDeclaresKindAndContentFormat(t *testing.T) {
	representation := markdown.New()
	if representation.Kind() != yacycrawlcontract.PageRepresentationKindMarkdown {
		t.Fatalf("representation = %q", representation.Kind())
	}
	if representation.ContentFormat() != contentformatgraph.FormatMarkdown {
		t.Fatalf("content format = %q", representation.ContentFormat())
	}
}
