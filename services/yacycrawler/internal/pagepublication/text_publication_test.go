package pagepublication_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagepublication"
)

func TestTextPublicationPublishes(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationText,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.text", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}
	publication := pagepublication.NewTextPublication(js, "yacy.crawl.page.text")
	representation := yacycrawlcontract.PageTextRepresentation{
		PageReference: sampleReference(),
		Text:          []byte("the quick brown fox"),
	}
	if err := publication.Publish(ctx, representation); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg := consumeOne(
		t,
		js,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationText),
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

func TestTextPublicationFullStreamIsRetryable(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationText,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.text.full", MaxMsgs: 1},
	); err != nil {
		t.Fatal(err)
	}
	publication := pagepublication.NewTextPublication(js, "yacy.crawl.page.text.full")
	representation := yacycrawlcontract.PageTextRepresentation{
		PageReference: sampleReference(),
		Text:          []byte("the quick brown fox"),
	}
	if err := publication.Publish(ctx, representation); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	err := publication.Publish(ctx, representation)
	var retryable crawlcapability.TransientPublicationError
	if err == nil || !errors.As(err, &retryable) {
		t.Fatalf("full stream should yield TransientPublicationError, got %v", err)
	}
}
