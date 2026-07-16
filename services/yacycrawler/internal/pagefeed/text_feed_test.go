package pagefeed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeed"
)

func TestTextFeedPublishesTheRenderedText(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationKindText,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.text", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}
	feed := pagefeed.NewTextFeed(js, "yacy.crawl.page.text", crawlcapability.PageContentFormatText)
	publication, err := feed.Derive(samplePage(), []byte("the quick brown fox"))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if err := feed.Publish(ctx, publication); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg := consumeOne(
		t,
		js,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindText),
	)
	page, err := yacycrawlcontract.UnmarshalPageTextRepresentation(msg)
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

func TestTextFeedFullStreamIsRetryable(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationKindText,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.text.full", MaxMsgs: 1},
	); err != nil {
		t.Fatal(err)
	}
	feed := pagefeed.NewTextFeed(
		js,
		"yacy.crawl.page.text.full",
		crawlcapability.PageContentFormatText,
	)
	publication, err := feed.Derive(samplePage(), []byte("the quick brown fox"))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if err := feed.Publish(ctx, publication); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	var retryable crawlcapability.TransientPublicationError
	if err := feed.Publish(ctx, publication); err == nil || !errors.As(err, &retryable) {
		t.Fatalf("full stream should yield TransientPublicationError, got %v", err)
	}
}

func TestTextFeedDeclaresRepresentationAndContentFormat(t *testing.T) {
	feed := pagefeed.NewTextFeed(nil, "yacy.crawl.page.text", crawlcapability.PageContentFormatText)
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindText {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if feed.ContentFormat() != crawlcapability.PageContentFormatText {
		t.Fatalf("content format = %q", feed.ContentFormat())
	}
}
