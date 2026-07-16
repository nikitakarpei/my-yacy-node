package pagefeed_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeed"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestRWIFeedPublishesTheIndexBuiltFromTheText(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationKindRWI,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.rwi", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}
	feed := pagefeed.NewRWIFeed(js, "yacy.crawl.page.rwi", crawlcapability.PageContentFormatText)
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
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindRWI),
	)
	chunk, err := yacycrawlcontract.UnmarshalPageRWIChunk(msg)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	metadata, ok := chunk.(yacycrawlcontract.PageRWIMetadataChunk)
	if !ok {
		t.Fatalf("first chunk = %T, want PageRWIMetadataChunk", chunk)
	}
	if metadata.CanonicalURL != samplePage().CanonicalURL {
		t.Fatalf("canonical url = %q", metadata.CanonicalURL)
	}
	if len(metadata.Metadata) != 1 {
		t.Fatalf("metadata rows = %d, want 1", len(metadata.Metadata))
	}
	row := metadata.Metadata[0]
	if row.Properties[yacymodel.URLMetaColDescription] !=
		yacymodel.EncodeBase64WireForm(samplePage().Title) {
		t.Fatalf("metadata row not framed as built: %+v", row.Properties)
	}
}

func TestRWIFeedChunksBoundPostings(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationKindRWI,
		yacycrawlcontract.CrawledPageStreamSpec{
			Subject: "yacy.crawl.page.rwi.bounded",
			MaxMsgs: 10,
		},
	); err != nil {
		t.Fatal(err)
	}
	words := make([]string, 2001)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	feed := pagefeed.NewRWIFeed(
		js,
		"yacy.crawl.page.rwi.bounded",
		crawlcapability.PageContentFormatText,
	)
	publication, err := feed.Derive(samplePage(), []byte(strings.Join(words, " ")))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if err := feed.Publish(ctx, publication); err != nil {
		t.Fatalf("publish: %v", err)
	}

	stream := yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindRWI)
	consumer, err := js.CreateOrUpdateConsumer(ctx, stream,
		jetstream.ConsumerConfig{AckPolicy: jetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatal(err)
	}
	for i, wantChunks := 0, 4; i < wantChunks; i++ {
		msg, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
		if err != nil {
			t.Fatalf("consume chunk %d: %v", i, err)
		}
		_ = msg.Ack()
		if _, err := yacycrawlcontract.UnmarshalPageRWIChunk(msg.Data()); err != nil {
			t.Fatalf("unmarshal chunk %d: %v", i, err)
		}
	}
}

func TestRWIFeedDeclaresRepresentationAndContentFormat(t *testing.T) {
	feed := pagefeed.NewRWIFeed(nil, "yacy.crawl.page.rwi", crawlcapability.PageContentFormatText)
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindRWI {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if feed.ContentFormat() != crawlcapability.PageContentFormatText {
		t.Fatalf("content format = %q", feed.ContentFormat())
	}
}
