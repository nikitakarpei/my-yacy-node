package pagefeed_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeed"
)

func TestMarkdownFeedPublishesTheRenderedMarkdown(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationKindMarkdown,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.markdown", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}
	feed := pagefeed.NewMarkdownFeed(
		js,
		"yacy.crawl.page.markdown",
		crawlcapability.PageContentFormatMarkdown,
	)
	publish, err := feed.Derive(samplePage(), []byte("# hi"))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if err := publish(ctx); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg := consumeOne(
		t,
		js,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindMarkdown),
	)
	page, err := yacycrawlcontract.UnmarshalPageMarkdownRepresentation(msg)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(page.Markdown) != "# hi" {
		t.Fatalf("markdown = %q", page.Markdown)
	}
}

func TestMarkdownFeedDeclaresRepresentationAndContentFormat(t *testing.T) {
	feed := pagefeed.NewMarkdownFeed(
		nil,
		"yacy.crawl.page.markdown",
		crawlcapability.PageContentFormatMarkdown,
	)
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindMarkdown {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if feed.ContentFormat() != crawlcapability.PageContentFormatMarkdown {
		t.Fatalf("content format = %q", feed.ContentFormat())
	}
}
